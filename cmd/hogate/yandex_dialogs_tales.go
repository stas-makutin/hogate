package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode/utf8"
)

type YandexDialogsTale struct {
	Name   string   `yaml:"name"`
	Keys   []string `yaml:"keys,omitempty"`
	Type   string   `yaml:"type"`
	Length uint32   `yaml:"length"`
	Parts  []string `yaml:"parts"`
}

type ydtFileType uint16

const (
	ydtTypeUnknown = ydtFileType(iota)
	ydtTypeFairyTale
	ydtTypeStory
	ydtTypeSong
	ydtTypeVerse
	ydtTypeJoke
)

type yandexDialogsTalesFile struct {
	name     string
	keys     []string
	fileType ydtFileType
	length   uint32
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

type yandexDialogsTalesItem struct {
	fileType ydtFileType
	index    int
}

type yandexDialogsTalesSlice struct {
	yandexDialogsTalesItem
	length int
}

type yandexDialogsTalesSelect struct {
	yandexDialogsTalesItem
	relative bool
}

var ydtFileTypes map[ydtFileType][]yandexDialogsTalesFile

var ydtRand *rand.Rand = nil

const ydtDefaultSliceLength = 3

const ydtMaxSessions = uint32(1000)

type yandexDialogsTalesSession struct {
	state    interface{}
	modified time.Time
}

var ydtSessions sync.Map
var ydtSessionCount uint32

func init() {
	ydtRand = rand.New(rand.NewSource(time.Now().UnixNano()))
}

func validateYandexDialogsTalesConfig(cfgError configError) {
	ydtFileTypes = make(map[ydtFileType][]yandexDialogsTalesFile)
	if config.YandexDialogs == nil || config.YandexDialogs.Tales == "" {
		return
	}
	var tales []YandexDialogsTale
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
	return 0, fmt.Errorf("Unrecognized tale type.")
}

func yandexDialogsTales(w http.ResponseWriter, r *http.Request) {
	var req YandexDialogsRequestEnvelope
	if !parseJsonRequest(&req, w, r) || req.Version != "1.0" {
		return
	}

	resp := YandexDialogsResponseEnvelope{
		Response: &YandexDialogsResponse{},
		Session: YandexDialogsResponseSession{
			SessionId: req.Session.SessionId,
			MessageId: req.Session.MessageId,
			UserId:    req.Session.UserId,
		},
		Version: "1.0",
	}

	if req.Request != nil && req.Request.Command == "test" {

		resp.Response.Text = req.Request.Command

	} else if status, _ := testAuthorization(r, scopeYandexDialogs); status != http.StatusOK {

		resp.Response.Text = "пожалуйста авторизируйтесь"
		resp.AccountLinking = &struct{}{}

	} else {
		state, sessionExists := yandexDialogsTalesGetSession(req.Session.SessionId)

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
			state = nil
			resp.Response.Text = "Пока"
			resp.Response.EndSession = true

		case ydtReactionOverview:
			fileType, _ := reactionData.(ydtFileType)
			state = yandexDialogsTalesReactionOverview(resp.Response, req.Session.SkillId, fileType)

		case ydtReactionSlice:
			if slice, ok := reactionData.(yandexDialogsTalesSlice); ok {
				state = yandexDialogsTalesReactionSlice(resp.Response, req.Session.SkillId, slice.fileType, slice.index, slice.length)
			} else {
				resp.Response.Text = errorText
			}

		case ydtReactionList:
			if list, ok := reactionData.([]yandexDialogsTalesItem); ok {
				state = yandexDialogsTalesReactionList(resp.Response, req.Session.SkillId, list)
			} else {
				resp.Response.Text = errorText
			}

		case ydtReactionNext:
			state = yandexDialogsTalesReactionNext(resp.Response, req.Session.SkillId, state)

		case ydtReactionPrevious:
			state = yandexDialogsTalesReactionPrevious(resp.Response, req.Session.SkillId, state)

		case ydtReactionRepeat:
			state = yandexDialogsTalesReactionRepeat(resp.Response, req.Session.SkillId, state)

		case ydtReactionSelect:
			if sel, ok := reactionData.(yandexDialogsTalesSelect); ok {
				state = yandexDialogsTalesReactionSelect(resp.Response, req.Session.SkillId, sel.fileType, sel.index, sel.relative, state)
			} else {
				resp.Response.Text = errorText
			}

		case ydtReactionRandom:
			if fileType, ok := reactionData.(ydtFileType); ok {
				state = yandexDialogsTalesReactionRandom(resp.Response, req.Session.SkillId, fileType)
			} else {
				resp.Response.Text = errorText
			}

		default: // ydtReactionNone
			if req.Session.New {
				state = nil
				resp.Response.Text = "Что бы вам рассказать?"
				resp.Response.Buttons = append(resp.Response.Buttons, YandexDialogsButton{Title: "А что есть?"})
			} else {
				state = yandexDialogsTalesReactionNotRecognized(resp.Response, req.Session.SkillId, state)
			}
		}

		yandexDialogsTalesSetSession(req.Session.SessionId, sessionExists, state)
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(resp)
}

func yandexDialogsTalesGetSession(sessionId string) (interface{}, bool) {
	var state interface{} = nil
	session, sessionExists := ydtSessions.Load(sessionId)
	if sessionExists {
		if s, ok := session.(yandexDialogsTalesSession); ok {
			state = s.state
		}
	}
	return state, sessionExists
}

func yandexDialogsTalesSetSession(sessionId string, sessionExists bool, state interface{}) {
	if state == nil {
		if sessionExists {
			atomic.AddUint32(&ydtSessionCount, ^uint32(0))
			ydtSessions.Delete(sessionId)
		}
	} else {
		if !sessionExists && atomic.LoadUint32(&ydtSessionCount) >= ydtMaxSessions {
			var sessionId interface{} = nil
			var modified time.Time
			ydtSessions.Range(func(k, v interface{}) bool {
				if s, ok := v.(yandexDialogsTalesSession); ok {
					if sessionId == nil || modified.After(s.modified) {
						sessionId = k
						modified = s.modified
					}
				} else {
					sessionId = k
					return false
				}
				return true
			})
			if sessionId != nil {
				atomic.AddUint32(&ydtSessionCount, ^uint32(0))
				ydtSessions.Delete(sessionId)
			}
		}
		ydtSessions.Store(sessionId, yandexDialogsTalesSession{state, time.Now()})
	}
}

func yandexDialogsTalesReactionNotRecognized(r *YandexDialogsResponse, skillId string, state interface{}) interface{} {
	r.Text = "Я вас не поняла, повторите пожалуйста."
	return state
}

func yandexDialogsTalesReactionOverview(r *YandexDialogsResponse, skillId string, fileType ydtFileType) interface{} {
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
			return yandexDialogsTalesReactionSlice(r, skillId, fileType, 0, ydtDefaultSliceLength)
		}
		t, _ := yandexDialogsTalesFileTypeName(fileType, 0)
		bt.WriteString("У меня пока нет никаких ")
		bt.WriteString(t)
		r.Buttons = append(r.Buttons, YandexDialogsButton{Title: "А что есть?"})
	}
	r.Text = bt.String()
	return nil
}

func yandexDialogsTalesReactionSlice(r *YandexDialogsResponse, skillId string, fileType ydtFileType, index, length int) interface{} {
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
			return yandexDialogsTalesSlice{yandexDialogsTalesItem{fileType, index}, length}
		}
	}

	t, _ := yandexDialogsTalesFileTypeName(fileType, 0)
	bt.WriteString("У меня больше нет ")
	bt.WriteString(t)
	r.Text = bt.String()
	r.Buttons = append(r.Buttons, YandexDialogsButton{Title: "А что есть?"})
	return nil
}

func yandexDialogsTalesReactionList(r *YandexDialogsResponse, skillId string, list []yandexDialogsTalesItem) interface{} {
	var bt strings.Builder

	bt.WriteString("У меня есть:")
	for i, item := range list {
		if f, ok := ydtFileTypes[item.fileType]; ok && item.index < len(f) {
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

func yandexDialogsTalesReactionNext(r *YandexDialogsResponse, skillId string, state interface{}) interface{} {
	if slice, ok := state.(yandexDialogsTalesSlice); ok {
		s := yandexDialogsTalesReactionSlice(r, skillId, slice.fileType, slice.index+slice.length, slice.length)
		if s == nil {
			return state
		}
		return s
	}
	return yandexDialogsTalesReactionNotRecognized(r, skillId, state)
}

func yandexDialogsTalesReactionPrevious(r *YandexDialogsResponse, skillId string, state interface{}) interface{} {
	if slice, ok := state.(yandexDialogsTalesSlice); ok {
		s := yandexDialogsTalesReactionSlice(r, skillId, slice.fileType, slice.index-slice.length, slice.length)
		if s == nil {
			return state
		}
		return s
	}
	return yandexDialogsTalesReactionNotRecognized(r, skillId, state)
}

func yandexDialogsTalesReactionRepeat(r *YandexDialogsResponse, skillId string, state interface{}) interface{} {
	if list, ok := state.([]yandexDialogsTalesItem); ok {
		return yandexDialogsTalesReactionList(r, skillId, list)
	} else if slice, ok := state.(yandexDialogsTalesSlice); ok {
		return yandexDialogsTalesReactionSlice(r, skillId, slice.fileType, slice.index, slice.length)
	} else if item, ok := state.(yandexDialogsTalesItem); ok {
		return yandexDialogsTalesReactionSelect(r, skillId, item.fileType, item.index, false, nil)
	}
	return yandexDialogsTalesReactionOverview(r, skillId, ydtTypeUnknown)
}

func yandexDialogsTalesReactionSelect(r *YandexDialogsResponse, skillId string, fileType ydtFileType, index int, relative bool, state interface{}) interface{} {
	if relative {
		if list, ok := state.([]yandexDialogsTalesItem); ok {
			if index >= 0 && index < len(list) {
				fileType = list[index].fileType
				index = list[index].index
			} else {
				index = -1
			}
		} else if slice, ok := state.(yandexDialogsTalesSlice); ok {
			fileType = slice.fileType
			index += slice.index
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
				btts.WriteString(fmt.Sprintf("<speaker audio='dialogs-upload/%v/%v.opus'>", skillId, id))
			}

			r.Text = bt.String()
			r.TTS = btts.String()
			r.Buttons = append(r.Buttons, YandexDialogsButton{Title: "Хватит"})

			return yandexDialogsTalesItem{fileType, index}
		}
	}

	r.Text = "Не нашла, попробуйте еще раз."
	return state
}

func yandexDialogsTalesReactionRandom(r *YandexDialogsResponse, skillId string, fileType ydtFileType) interface{} {
	if len(ydtFileTypes) <= 0 {
		return yandexDialogsTalesReactionOverview(r, skillId, ydtTypeUnknown)
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
				return yandexDialogsTalesReactionOverview(r, skillId, ydtTypeUnknown)
			}
		}
	}

	if f, ok := ydtFileTypes[fileType]; ok && len(f) > 0 {
		index := ydtRand.Intn(len(f))
		return yandexDialogsTalesReactionSelect(r, skillId, fileType, index, false, nil)
	}

	return yandexDialogsTalesReactionOverview(r, skillId, fileType)
}

var ydtwmDone = map[string]struct{}{
	"хватит": struct{}{}, "выйти": struct{}{}, "выйди": struct{}{}, "закончи": struct{}{}, "закончить": struct{}{},
	"прекрати": struct{}{}, "прекратить": struct{}{}, "остановись": struct{}{}, "стоп": struct{}{},
}

var ydtwmNext = map[string]struct{}{
	"дальше": struct{}{}, "еще": struct{}{}, "ещё": struct{}{}, "следующий": struct{}{}, "следующие": struct{}{}, "следующая": struct{}{},
}

var ydtwmPrevious = map[string]struct{}{
	"перед": struct{}{}, "предыдущие": struct{}{}, "предыдущая": struct{}{}, "предыдущий": struct{}{},
}

var ydtwmRepeat = map[string]struct{}{
	"повтори": struct{}{}, "повторить": struct{}{},
}

var ydtwmUntil = map[string]struct{}{
	"по": struct{}{}, "до": struct{}{},
}

var ydtwmPlay = map[string]struct{}{
	"расскажи": struct{}{}, "рассказать": struct{}{}, "рассказывай": struct{}{},
	"давай": struct{}{},
	"спой":  struct{}{}, "спеть": struct{}{},
}

var ydtwmRandom = map[string]struct{}{
	"что-нибудь": struct{}{}, "случайно": struct{}{}, "случайную": struct{}{}, "случайная": struct{}{}, "случайный": struct{}{},
	"любой": struct{}{}, "любую": struct{}{}, "любое": struct{}{},
	"какую-нибудь": struct{}{}, "какой-нибудь": struct{}{}, "какое-нибудь": struct{}{},
}

var ydtwmOverview = map[string]struct{}{
	"что": struct{}{}, "какая": struct{}{}, "какое": struct{}{}, "какой": struct{}{}, "какие": struct{}{}, "есть": struct{}{},
	"список": struct{}{}, "чем": struct{}{}, "чём": struct{}{}, "можешь": struct{}{},
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
		return ydtReactionSlice, yandexDialogsTalesSlice{yandexDialogsTalesItem{fileType, firstNumber}, secondNumber}
	}
	if firstNumber > 0 {
		return ydtReactionSelect, yandexDialogsTalesSelect{yandexDialogsTalesItem{fileType, firstNumber - 1}, true}
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
				if strings.Index(tokens, k) >= 0 {
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
				found = append(found, yandexDialogsTalesItem{ft, i})
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
	var m int = 0
	if !(n > 10 && n < 15) {
		m = n % 10
	}
	switch {
	case m == 1:
		switch {
		case r > 0:
			return "одна"
		case r < 0:
			return "один"
		default:
			return "одно"
		}
	case m == 2:
		switch {
		case r > 0:
			return "две"
		default:
			return "два"
		}
	}
	return strconv.Itoa(n)
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
