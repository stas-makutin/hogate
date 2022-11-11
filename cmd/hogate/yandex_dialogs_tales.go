package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

type yandexDialogsTale struct {
	Name   string   `yaml:"name"`
	Keys   []string `yaml:"keys,omitempty"`
	Type   string   `yaml:"type"`
	Length int32    `yaml:"length"`
	Parts  []string `yaml:"parts"`
}

type yandexDialogsTalesFile struct {
	name     string
	keys     []string
	fileType ydtFileType
	length   int32
	ids      []string
}

type ydtReaction uint16

const (
	ydtReactionNone = ydtReaction(iota)
	ydtReactionOverview
	ydtReactionSlice
	ydtReactionList
	ydtReactionNext
	ydtReactionPrevious
	ydtReactionRepeat
	ydtReactionSelect
	ydtReactionRandom
	ydtReactionDone
)

var ydtFileTypes map[ydtFileType][]yandexDialogsTalesFile

var ydtRand *rand.Rand = nil

const ydtDefaultSliceLength = 5

const ydtMaxSessions = int32(1000)

type yandexDialogsTalesSession struct {
	state    interface{}
	modified time.Time
}

var ydtSessions sync.Map
var ydtSessionCount int32

func init() {
	ydtRand = rand.New(rand.NewSource(time.Now().UnixNano()))
}

func validateYandexDialogsTalesConfig(cfgError configError) {
	ydtFileTypes = make(map[ydtFileType][]yandexDialogsTalesFile)
	if config.YandexDialogs == nil || config.YandexDialogs.Tales == "" {
		return
	}
	var tales []yandexDialogsTale
	if err := loadSubConfig(config.YandexDialogs.Tales, &tales); err != nil {
		cfgError(fmt.Sprintf("yandexDialogs.tales, unable to load configuration file '%v': %v", config.YandexDialogs.Tales, err))
		return
	}

	for i, tale := range tales {
		taleError := func(msg string) {
			cfgError(fmt.Sprintf("yandexDialogs.tales, tale %v: %v", i, msg))
		}

		fileType, err := parseYandexDialogsTaleType(tale.Type)
		if err != nil {
			taleError(fmt.Sprintf("unknown type '%v'.", tale.Type))
			continue
		}

		file := yandexDialogsTalesFile{
			name:     tale.Name,
			keys:     tale.Keys,
			fileType: fileType,
			length:   tale.Length,
			ids:      tale.Parts,
		}

		if len(file.keys) <= 0 {
			for _, k := range strings.Fields(strings.ToLower(file.name)) {
				if utf8.RuneCountInString(k) > 2 {
					file.keys = append(file.keys, k)
				}
			}
		}

		if tales, ok := ydtFileTypes[fileType]; ok {
			ydtFileTypes[fileType] = append(tales, file)
		} else {
			ydtFileTypes[fileType] = []yandexDialogsTalesFile{file}
		}
	}
}

func parseYandexDialogsTaleType(t string) (ydtFileType, error) {
	switch strings.ToLower(t) {
	case "fairytale":
		return ydtTypeFairyTale, nil
	case "story":
		return ydtTypeStory, nil
	case "song":
		return ydtTypeSong, nil
	case "verse":
		return ydtTypeVerse, nil
	case "joke":
		return ydtTypeJoke, nil
	}
	return 0, fmt.Errorf("unrecognized tale type")
}

func yandexDialogsTales(w http.ResponseWriter, r *http.Request) {
	var req YandexDialogsRequestEnvelope
	if !parseJSONRequest(&req, w, r) || req.Version != "1.0" {
		return
	}

	resp := YandexDialogsResponseEnvelope{
		Response: &YandexDialogsResponse{},
		Session: YandexDialogsResponseSession{
			SessionID: req.Session.SessionID,
			MessageID: req.Session.MessageID,
			UserID:    req.Session.UserID,
		},
		Version: "1.0",
	}

	if req.Request != nil && req.Request.Command == "test" {

		resp.Response.Text = req.Request.Command

	} else if status, _ := testAuthorization(r, scopeYandexDialogs); status != http.StatusOK {

		resp.Response.Text = "пожалуйста авторизируйтесь"
		resp.AccountLinking = &struct{}{}

	} else {
		state := yandexDialogsTalesGetSession(req.State)

		if req.AccountLinking != nil {
			req.Session.New = true
		}

		errorText := "Что-то пошло не так"

		reaction, reactionData := ydtReactionNone, interface{}(nil)
		if req.Request != nil {
			reaction, reactionData = yandexDialogsTalesReaction(*req.Request)
		}
		switch reaction {
		case ydtReactionDone:
			if _, ok := state.(yandexDialogsTalesItem); ok {
				state = nil
				resp.Response.Text = "Рассказать что-нибудь еще?"
				resp.Response.Buttons = append(resp.Response.Buttons, YandexDialogsButton{Title: "А что есть?"})
			} else {
				state = nil
				resp.Response.Text = "Пока"
				resp.Response.EndSession = true
			}
		case ydtReactionOverview:
			fileType, _ := reactionData.(ydtFileType)
			state = yandexDialogsTalesReactionOverview(resp.Response, req.Session.SkillID, fileType)

		case ydtReactionSlice:
			if slice, ok := reactionData.(yandexDialogsTalesSlice); ok {
				state = yandexDialogsTalesReactionSlice(resp.Response, req.Session.SkillID, slice.fileType, int(slice.index), int(slice.length))
			} else {
				resp.Response.Text = errorText
			}

		case ydtReactionList:
			if list, ok := reactionData.([]yandexDialogsTalesItem); ok {
				state = yandexDialogsTalesReactionList(resp.Response, req.Session.SkillID, list)
			} else {
				resp.Response.Text = errorText
			}

		case ydtReactionNext:
			state = yandexDialogsTalesReactionNext(resp.Response, req.Session.SkillID, state)

		case ydtReactionPrevious:
			state = yandexDialogsTalesReactionPrevious(resp.Response, req.Session.SkillID, state)

		case ydtReactionRepeat:
			state = yandexDialogsTalesReactionRepeat(resp.Response, req.Session.SkillID, state)

		case ydtReactionSelect:
			if sel, ok := reactionData.(yandexDialogsTalesSelect); ok {
				state = yandexDialogsTalesReactionSelect(resp.Response, req.Session.SkillID, sel.fileType, int(sel.index), sel.relative, state)
			} else {
				resp.Response.Text = errorText
			}

		case ydtReactionRandom:
			if fileType, ok := reactionData.(ydtFileType); ok {
				state = yandexDialogsTalesReactionRandom(resp.Response, req.Session.SkillID, fileType)
			} else {
				resp.Response.Text = errorText
			}

		default: // ydtReactionNone
			if req.Session.New {
				state = nil
				resp.Response.Text = "Что бы вам рассказать?"
				resp.Response.Buttons = append(resp.Response.Buttons, YandexDialogsButton{Title: "А что есть?"})
			} else {
				state = yandexDialogsTalesReactionNotRecognized(resp.Response, req.Session.SkillID, state)
			}
		}

		resp.SessionState = yandexDialogsTalesSetSession(state)
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(resp)
}

func yandexDialogsTalesGetSession(state *YandexDialogsRequestState) interface{} {
	if state != nil {
		if s, ok := state.Session["value"]; ok {
			if st, err := decodeState(s); err == nil {
				return st
			}
		}
	}
	return nil
}

func yandexDialogsTalesSetSession(state interface{}) interface{} {
	if state != nil {
		if s, err := encodeState(state); err == nil {
			return map[string]string{
				"value": s,
			}
		}
	}
	return nil
}

func yandexDialogsTalesReactionNotRecognized(r *YandexDialogsResponse, skillID string, state interface{}) interface{} {
	r.Text = "Я вас не поняла, повторите пожалуйста."
	return state
}

func yandexDialogsTalesReactionOverview(r *YandexDialogsResponse, skillID string, fileType ydtFileType) interface{} {
	var bt strings.Builder
	if fileType == ydtTypeUnknown {
		none := true
		bt.WriteString("У меня есть ")
		for k, v := range ydtFileTypes {
			c := len(v)
			if c <= 0 {
				continue
			}
			if !none {
				bt.WriteString(", ")
			}
			none = false

			t, g := yandexDialogsTalesFileTypeName(k, c)
			bt.WriteString(yandexDialogsTalesNumber(c, g))
			bt.WriteString(" ")
			bt.WriteString(t)

			t, _ = yandexDialogsTalesFileTypeName(k, 0)
			r.Buttons = append(r.Buttons, YandexDialogsButton{Title: "Список " + t})
		}

		if none {
			bt.Reset()
			bt.WriteString("Пока у меня ничего нет")
			r.Buttons = append(r.Buttons, YandexDialogsButton{Title: "Выйти"})
		}
	} else {
		if _, ok := ydtFileTypes[fileType]; ok {
			return yandexDialogsTalesReactionSlice(r, skillID, fileType, 0, ydtDefaultSliceLength)
		}
		t, _ := yandexDialogsTalesFileTypeName(fileType, 0)
		bt.WriteString("У меня пока нет никаких ")
		bt.WriteString(t)
		r.Buttons = append(r.Buttons, YandexDialogsButton{Title: "А что есть?"})
	}
	r.Text = bt.String()
	return nil
}

func yandexDialogsTalesReactionSlice(r *YandexDialogsResponse, skillID string, fileType ydtFileType, index, length int) interface{} {
	var bt strings.Builder
	if f, ok := ydtFileTypes[fileType]; ok {
		c := len(f)
		if index < 0 {
			index = 0
		}
		if c > 0 && index >= 0 && index < c && length > 0 {
			l := length
			if index+l > c {
				l = c - index
			}
			t, g := "", -1
			if l == 1 {
				t, g = yandexDialogsTalesFileTypeName(fileType, 1)
				bt.WriteString(yandexDialogsTalesSequence(index+1, g, 0))
				bt.WriteString(" ")
				bt.WriteString(t)
			} else {
				t, g = "стишки", -1
				if fileType != ydtTypeVerse {
					t, g = yandexDialogsTalesFileTypeName(fileType, 2)
				}
				bt.WriteString(t)
				bt.WriteString(" с ")
				bt.WriteString(yandexDialogsTalesSequence(index+1, g, 2))
				bt.WriteString(" по ")
				bt.WriteString(yandexDialogsTalesSequence(index+l, g, 1))
			}
			bt.WriteString(":")

			for i := index; i < index+l; i++ {
				bt.WriteString("\n")
				bt.WriteString(f[i].name)
				r.Buttons = append(r.Buttons, YandexDialogsButton{Title: yandexDialogsTalesSequence(i-index+1, g, 0)})
			}
			if index > 0 {
				r.Buttons = append(r.Buttons, YandexDialogsButton{Title: "предыдущие"})
			}
			if index+l < c {
				r.Buttons = append(r.Buttons, YandexDialogsButton{Title: "следующие"})
			}

			r.Text = bt.String()
			return yandexDialogsTalesSlice{yandexDialogsTalesItem{fileType, int32(index)}, int32(length)}
		}
	}

	t, _ := yandexDialogsTalesFileTypeName(fileType, 0)
	bt.WriteString("У меня больше нет ")
	bt.WriteString(t)
	r.Text = bt.String()
	r.Buttons = append(r.Buttons, YandexDialogsButton{Title: "А что есть?"})
	return nil
}

func yandexDialogsTalesReactionList(r *YandexDialogsResponse, skillID string, list []yandexDialogsTalesItem) interface{} {
	var bt strings.Builder

	bt.WriteString("У меня есть:")
	for i, item := range list {
		if f, ok := ydtFileTypes[item.fileType]; ok && int(item.index) < len(f) {
			t, g := yandexDialogsTalesFileTypeName(item.fileType, 1)
			bt.WriteString("\n")
			bt.WriteString(t)
			bt.WriteString(" ")
			bt.WriteString(f[item.index].name)
			r.Buttons = append(r.Buttons, YandexDialogsButton{Title: yandexDialogsTalesSequence(i+1, g, 0)})
		}
	}

	r.Text = bt.String()
	return list
}

func yandexDialogsTalesReactionNext(r *YandexDialogsResponse, skillID string, state interface{}) interface{} {
	if slice, ok := state.(yandexDialogsTalesSlice); ok {
		s := yandexDialogsTalesReactionSlice(r, skillID, slice.fileType, int(slice.index+slice.length), int(slice.length))
		if s == nil {
			return state
		}
		return s
	}
	return yandexDialogsTalesReactionNotRecognized(r, skillID, state)
}

func yandexDialogsTalesReactionPrevious(r *YandexDialogsResponse, skillID string, state interface{}) interface{} {
	if slice, ok := state.(yandexDialogsTalesSlice); ok {
		s := yandexDialogsTalesReactionSlice(r, skillID, slice.fileType, int(slice.index-slice.length), int(slice.length))
		if s == nil {
			return state
		}
		return s
	}
	return yandexDialogsTalesReactionNotRecognized(r, skillID, state)
}

func yandexDialogsTalesReactionRepeat(r *YandexDialogsResponse, skillID string, state interface{}) interface{} {
	if list, ok := state.([]yandexDialogsTalesItem); ok {
		return yandexDialogsTalesReactionList(r, skillID, list)
	} else if slice, ok := state.(yandexDialogsTalesSlice); ok {
		return yandexDialogsTalesReactionSlice(r, skillID, slice.fileType, int(slice.index), int(slice.length))
	} else if item, ok := state.(yandexDialogsTalesItem); ok {
		return yandexDialogsTalesReactionSelect(r, skillID, item.fileType, int(item.index), false, nil)
	}
	return yandexDialogsTalesReactionOverview(r, skillID, ydtTypeUnknown)
}

func yandexDialogsTalesReactionSelect(r *YandexDialogsResponse, skillID string, fileType ydtFileType, index int, relative bool, state interface{}) interface{} {
	if relative {
		if list, ok := state.([]yandexDialogsTalesItem); ok {
			if index >= 0 && index < len(list) {
				fileType = list[index].fileType
				index = int(list[index].index)
			} else {
				index = -1
			}
		} else if slice, ok := state.(yandexDialogsTalesSlice); ok {
			fileType = slice.fileType
			index += int(slice.index)
		} else {
			index--
		}
	}
	if index >= 0 {
		if f, ok := ydtFileTypes[fileType]; ok && index < len(f) {
			var bt strings.Builder
			var btts strings.Builder

			t, _ := yandexDialogsTalesFileTypeName(fileType, 1)
			bt.WriteString(t)
			bt.WriteString(" ")
			bt.WriteString(f[index].name)

			for _, id := range f[index].ids {
				btts.WriteString(fmt.Sprintf("<speaker audio='dialogs-upload/%v/%v.opus'>", skillID, id))
			}
			btts.WriteString("Рассказать что-нибудь еще?")

			r.Text = bt.String()
			r.TTS = btts.String()
			r.Buttons = append(r.Buttons, YandexDialogsButton{Title: "Хватит"})
			r.Buttons = append(r.Buttons, YandexDialogsButton{Title: "А что есть?"})

			return yandexDialogsTalesItem{fileType, int32(index)}
		}
	}

	r.Text = "Не нашла, попробуйте еще раз."
	return state
}

func yandexDialogsTalesReactionRandom(r *YandexDialogsResponse, skillID string, fileType ydtFileType) interface{} {
	if len(ydtFileTypes) <= 0 {
		return yandexDialogsTalesReactionOverview(r, skillID, ydtTypeUnknown)
	}

	if fileType == ydtTypeUnknown {
		fileType = ydtFileType(ydtRand.Intn(int(ydtTypeJoke)) + 1)
		o := fileType
		for {
			if _, ok := ydtFileTypes[fileType]; ok {
				break
			}
			if fileType == ydtTypeJoke {
				fileType = ydtTypeFairyTale
			} else {
				fileType++
			}
			if fileType == o {
				return yandexDialogsTalesReactionOverview(r, skillID, ydtTypeUnknown)
			}
		}
	}

	if f, ok := ydtFileTypes[fileType]; ok && len(f) > 0 {
		index := ydtRand.Intn(len(f))
		return yandexDialogsTalesReactionSelect(r, skillID, fileType, index, false, nil)
	}

	return yandexDialogsTalesReactionOverview(r, skillID, fileType)
}

var ydtwmDone = map[string]struct{}{
	"хватит": {}, "выйти": {}, "выйди": {}, "закончи": {}, "закончить": {},
	"прекрати": {}, "прекратить": {}, "остановись": {}, "стоп": {},
}

var ydtwmNext = map[string]struct{}{
	"дальше": {}, "еще": {}, "ещё": {}, "следующий": {}, "следующие": {}, "следующая": {},
}

var ydtwmPrevious = map[string]struct{}{
	"перед": {}, "предыдущие": {}, "предыдущая": {}, "предыдущий": {},
}

var ydtwmRepeat = map[string]struct{}{
	"повтори": {}, "повторить": {},
}

var ydtwmUntil = map[string]struct{}{
	"по": {}, "до": {},
}

var ydtwmPlay = map[string]struct{}{
	"расскажи": {}, "рассказать": {}, "рассказывай": {},
	"давай": {},
	"спой":  {}, "спеть": {},
}

var ydtwmRandom = map[string]struct{}{
	"что-нибудь": {}, "случайно": {}, "случайную": {}, "случайная": {}, "случайный": {},
	"любой": {}, "любую": {}, "любое": {},
	"какую-нибудь": {}, "какой-нибудь": {}, "какое-нибудь": {},
}

var ydtwmOverview = map[string]struct{}{
	"что": {}, "какая": {}, "какое": {}, "какой": {}, "какие": {}, "есть": {},
	"список": {}, "чем": {}, "чём": {}, "можешь": {},
}

var ydtwmFileType = map[string]ydtFileType{
	"сказка": ydtTypeFairyTale, "сказки": ydtTypeFairyTale, "сказке": ydtTypeFairyTale, "сказкой": ydtTypeFairyTale, "сказку": ydtTypeFairyTale, "сказок": ydtTypeFairyTale, "сказками": ydtTypeFairyTale,
	"история": ydtTypeStory, "истории": ydtTypeStory, "историей": ydtTypeStory, "историй": ydtTypeStory, "историю": ydtTypeStory,
	"песня": ydtTypeSong, "песни": ydtTypeSong, "песне": ydtTypeSong, "песней": ydtTypeSong, "песен": ydtTypeSong, "песню": ydtTypeSong,
	"стишок": ydtTypeVerse, "стишка": ydtTypeVerse, "стишку": ydtTypeVerse, "стишком": ydtTypeVerse, "стишков": ydtTypeVerse, "стишки": ydtTypeVerse,
	"шутка": ydtTypeJoke, "шутки": ydtTypeJoke, "шутке": ydtTypeJoke, "шуткой": ydtTypeJoke, "шуток": ydtTypeJoke, "шутку": ydtTypeSong,
}

func yandexDialogsTalesReaction(r YandexDialogsRequest) (ydtReaction, interface{}) {
	if r.Nlu == nil && len(r.Nlu.Tokens) <= 0 {
		return ydtReactionNone, nil
	}

	var overviewState bool = false
	var randomState bool = false
	var untilState bool = false
	var playState bool = false
	var firstNumber int = 0
	var secondNumber int = 0
	var fileType ydtFileType = ydtTypeUnknown

	var tb strings.Builder
	for _, t := range r.Nlu.Tokens {
		t = strings.ToLower(t)
		if ft, ok := ydtwmFileType[t]; ok {
			fileType = ft
			continue
		}

		if tb.Len() > 0 {
			tb.WriteString(" ")
		}
		tb.WriteString(t)

		if _, ok := ydtwmDone[t]; ok {
			return ydtReactionDone, nil
		}
		if _, ok := ydtwmNext[t]; ok {
			return ydtReactionNext, nil
		}
		if _, ok := ydtwmPrevious[t]; ok {
			return ydtReactionPrevious, nil
		}
		if _, ok := ydtwmRepeat[t]; ok {
			return ydtReactionRepeat, nil
		}

		if _, ok := ydtwmOverview[t]; ok {
			overviewState = true
		} else if _, ok := ydtwmRandom[t]; ok {
			randomState = true
		} else if _, ok := ydtwmUntil[t]; ok {
			untilState = true
		} else if _, ok := ydtwmPlay[t]; ok {
			playState = true
		} else if n, err := strconv.Atoi(t); err == nil && n > 0 {
			if firstNumber == 0 {
				firstNumber = n
			} else if secondNumber == 0 {
				secondNumber = n
			}
		}
	}

	if randomState {
		return ydtReactionRandom, fileType
	}
	if secondNumber > 0 {
		if untilState {
			// c <first> по <second>
			firstNumber--
			secondNumber = secondNumber - firstNumber
		} else {
			// <first> c <second> пять с второй
			firstNumber, secondNumber = secondNumber-1, firstNumber
		}
		return ydtReactionSlice, yandexDialogsTalesSlice{yandexDialogsTalesItem{fileType, int32(firstNumber)}, int32(secondNumber)}
	}
	if firstNumber > 0 {
		return ydtReactionSelect, yandexDialogsTalesSelect{yandexDialogsTalesItem{fileType, int32(firstNumber - 1)}, true}
	}
	if playState {
		return ydtReactionSelect, yandexDialogsTalesSelect{yandexDialogsTalesItem{fileType, 0}, true}
	}
	if overviewState || (len(r.Nlu.Tokens) == 1 && fileType != ydtTypeUnknown) {
		return ydtReactionOverview, fileType
	}

	tokens := tb.String()
	found := []yandexDialogsTalesItem{}
	bestMk := 0
	for ft, fs := range ydtFileTypes {
		if fileType != ydtTypeUnknown && fileType != ft {
			continue
		}
		for i, f := range fs {
			mk := 0
			for _, k := range f.keys {
				if strings.Contains(tokens, k) {
					mk++
				}
				if tokens == k {
					mk++
				}
			}
			if mk > 0 {
				if mk > bestMk {
					found = nil
					bestMk = mk
				} else if mk < bestMk {
					continue
				}
				found = append(found, yandexDialogsTalesItem{ft, int32(i)})
			}
		}
	}
	l := len(found)
	if l == 1 {
		return ydtReactionSelect, yandexDialogsTalesSelect{found[0], false}
	} else if l > 1 {
		return ydtReactionList, found
	}

	return ydtReactionNone, nil
}

func yandexDialogsTalesFileTypeName(fileType ydtFileType, count int) (text string, kind int) {
	var m int = 0
	if !(count > 10 && count < 15) {
		m = count % 10
	}
	kind = 1
	switch fileType {
	case ydtTypeFairyTale:
		if m == 1 {
			text = "сказка"
		} else if m > 1 && m < 5 {
			text = "сказки"
		} else {
			text = "сказок"
		}
	case ydtTypeStory:
		if m == 1 {
			text = "история"
		} else if m > 1 && m < 5 {
			text = "ист+ории"
		} else {
			text = "историй"
		}
	case ydtTypeSong:
		if m == 1 {
			text = "песня"
		} else if m > 1 && m < 5 {
			text = "песни"
		} else {
			text = "песен"
		}
	case ydtTypeVerse:
		kind = -1
		if m == 1 {
			text = "стишок"
		} else if m > 1 && m < 5 {
			text = "стишка"
		} else {
			text = "стишков"
		}
	case ydtTypeJoke:
		if m == 1 {
			text = "шутка"
		} else if m > 1 && m < 5 {
			text = "шутки"
		} else {
			text = "шуток"
		}
	}
	return
}

func yandexDialogsTalesNumber(n, r int) string {
	if n > 999 || n <= 0 {
		return strconv.Itoa(n)
	}

	var names = map[int]map[int]string{
		0: {
			1: "один", 2: "два", 3: "три", 4: "четрые", 5: "пять", 6: "шесть", 7: "семь", 8: "восемь", 9: "девять",
			10: "десять", 11: "одинадцать", 12: "двенадцать", 13: "тринадцать", 14: "четырнадцать",
			15: "пятнадцать", 16: "шестнадцать", 17: "семнадцать", 18: "восемнадцать", 19: "девятнадцать",
		},
		1: {2: "двадцать", 3: "тридцать", 4: "сорок", 5: "пятьдесят", 6: "шестьдесят", 7: "семьдесят", 8: "восемьдесят", 9: "девяносто"},
		2: {1: "сто", 2: "двести", 3: "триста", 4: "четыреста", 5: "пятьсот", 6: "шестьсот", 7: "семьсот", 8: "восемьсот", 9: "девятьсот"},
	}

	text := ""
	l := 0
	d := 100
	for n > 0 {
		name := ""

		m := n % d
		if l == 0 {
			nl := l
			if m < 20 {
				n /= 10
				nl++
			} else {
				m = n % 10
			}
			if m > 0 {
				name = names[l][m]
				switch m {
				case 1:
					if r > 0 {
						name = "одна"
					} else if r == 0 {
						name = "одно"
					}
				case 2:
					if r > 0 {
						name = "две"
					}
				}
			}
			l = nl
		} else if m > 0 {
			name = names[l][m]
		}

		if name != "" {
			if text != "" {
				text = name + " " + text
			} else {
				text = name
			}
		}

		n /= 10
		d = 10
		l++
	}

	return text
}

func yandexDialogsTalesSequence(n, r, c int) (text string) {
	text = strconv.Itoa(n) + "-"
	m := n % 20
	switch {
	case r > 0:
		switch c {
		default: // какая?
			switch {
			case m == 3:
				text += "ья"
			default:
				text += "ая"
			}
		case 1: // какую?
			switch {
			case m == 3:
				text += "ью"
			default:
				text += "ую"
			}
		case 2: // какой?
			switch {
			case m == 3:
				text += "ей"
			default:
				text += "ой"
			}
		}
	case r < 0:
		switch c {
		default: // какой?
			switch {
			case n == 0 || m == 2:
				text += "ой"
			case m == 3:
				text += "ий"
			default:
				text += "ый"
			}
		case 2: // какого?
			text += "го"
		}
	default:
		switch c {
		default: // какое?
			switch {
			case m == 3:
				text += "ее"
			default:
				text += "ое"
			}
		case 2: // какого?
			text += "го"
		}
	}
	return
}
