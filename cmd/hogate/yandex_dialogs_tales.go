package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

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
	ttsName  string
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

const ydtMaxSessions = uint32(1000)

type yandexDialogsTalesSession struct {
	state    interface{}
	modified time.Time
}

var ydtSessions sync.Map
var ydtSessionCount uint32

func validateYandexDialogsTalesConfig(cfgError configError) {
	ydtFileTypes = make(map[ydtFileType][]yandexDialogsTalesFile)
	if config.YandexDialogs == nil {
		return
	}

	for i, tale := range config.YandexDialogs.Tales {
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
			ttsName:  tale.TtsName,
			keys:     tale.Keys,
			fileType: fileType,
			length:   tale.Length,
			ids:      tale.Parts,
		}

		if len(file.keys) <= 0 {
			file.keys = strings.Fields(strings.ToLower(file.name))
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

	state, sessionExists := yandexDialogsTalesGetSession(req.Session.SessionId)

	resp := YandexDialogsResponseEnvelope{
		Session: YandexDialogsResponseSession{
			SessionId: req.Session.SessionId,
			MessageId: req.Session.MessageId,
			UserId:    req.Session.UserId,
		},
		Version: "1.0",
	}

	errorText := "Что-то пошло не так"

	reaction, reactionData := yandexDialogsTalesReaction(req.Request)
	switch reaction {
	case ydtReactionDone:
		state = nil
		resp.Response.Text = "Пока"
		resp.Response.EndSession = true

	case ydtReactionOverview:
		fileType, _ := reactionData.(ydtFileType)
		state = yandexDialogsTalesReactionOverview(&resp.Response, fileType)

	case ydtReactionSlice:
		if slice, ok := reactionData.(yandexDialogsTalesSlice); ok {
			state = yandexDialogsTalesReactionSlice(&resp.Response, slice.fileType, slice.index, slice.length)
		} else {
			resp.Response.Text = errorText
		}

	case ydtReactionList:
		if list, ok := reactionData.([]yandexDialogsTalesItem); ok {
			state = yandexDialogsTalesReactionList(&resp.Response, list)
		} else {
			resp.Response.Text = errorText
		}

	case ydtReactionNext:

	case ydtReactionPrevious:

	case ydtReactionSelect:

	case ydtReactionRandom:

	default: // ydtReactionNone
		if req.Session.New {
			state = nil
			resp.Response.Text = "Что бы вам рассказать?"
		} else {
			resp.Response.Text = "Я вас не поняла, повторите пожалуйста."
		}
	}

	yandexDialogsTalesSetSession(req.Session.SessionId, sessionExists, state)

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

func yandexDialogsTalesReactionOverview(r *YandexDialogsResponse, fileType ydtFileType) interface{} {
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

			t, r := yandexDialogsTalesFileTypeName(k, c)
			bt.WriteString(yandexDialogsTalesNumber(c, r))
			bt.WriteString(" ")
			bt.WriteString(t)
		}

		if none {
			bt.Reset()
			bt.WriteString("Пока у меня ничего нет")
		}
	} else {
		return yandexDialogsTalesReactionSlice(r, fileType, 0, 3)
	}
	r.Text = bt.String()
	return nil
}

func yandexDialogsTalesReactionSlice(r *YandexDialogsResponse, fileType ydtFileType, index, length int) interface{} {
	var bt strings.Builder
	if f, ok := ydtFileTypes[fileType]; ok && len(f) > 0 {
		c := len(f)
		if index >= c {

		}
		/*

			t, r := yandexDialogsTalesFileTypeName(*fileType, c)
			bt.WriteString("У меня есть ")

			bt.WriteString(yandexDialogsTalesNumber(c, r))
			bt.WriteString(" ")
			bt.WriteString(t)
		*/

		// TODO - list first 3

	} else {
		t, _ := yandexDialogsTalesFileTypeName(fileType, 0)
		bt.WriteString("Пока у меня нет ")
		bt.WriteString(t)
	}
	return nil
}

func yandexDialogsTalesReactionList(r *YandexDialogsResponse, list []yandexDialogsTalesItem) interface{} {
	return nil
}

var ydtwmDone = map[string]struct{}{
	"хватит": struct{}{}, "выйти": struct{}{}, "выйди": struct{}{}, "закончи": struct{}{}, "закончить": struct{}{},
	"прекрати": struct{}{}, "прекратить": struct{}{}, "остановись": struct{}{}, "стоп": struct{}{},
}

var ydtwmNext = map[string]struct{}{
	"дальше": struct{}{}, "еще": struct{}{}, "ещё": struct{}{}, "следующий": struct{}{}, "следующие": struct{}{}, "следующая": struct{}{},
}

var ydtwmPrevious = map[string]struct{}{
	"перед": struct{}{}, "повтори": struct{}{}, "повторить": struct{}{}, "предыдущий": struct{}{}, "предыдущие": struct{}{}, "предыдущая": struct{}{},
}

var ydtwmRandom = map[string]struct{}{
	"что-нибудь": struct{}{}, "случайно": struct{}{}, "случайную": struct{}{}, "случайная": struct{}{}, "случайный": struct{}{},
	"любой": struct{}{}, "любую": struct{}{}, "любое": struct{}{},
}

var ydtwmOverview = map[string]struct{}{
	"что": struct{}{}, "какая": struct{}{}, "какое": struct{}{}, "какой": struct{}{}, "какие": struct{}{}, "есть": struct{}{},
	"список": struct{}{}, "чем": struct{}{}, "чём": struct{}{}, "можешь": struct{}{},
}

var ydtwmFileType = map[string]ydtFileType{
	"сказка": ydtTypeFairyTale, "сказки": ydtTypeFairyTale, "сказке": ydtTypeFairyTale, "сказкой": ydtTypeFairyTale, "сказку": ydtTypeFairyTale, "сказок": ydtTypeFairyTale, "сказками": ydtTypeFairyTale,
	"история": ydtTypeStory, "истории": ydtTypeStory, "историей": ydtTypeStory, "историй": ydtTypeStory,
	"песня": ydtTypeSong, "песни": ydtTypeSong, "песне": ydtTypeSong, "песней": ydtTypeSong, "песен": ydtTypeSong,
	"стишок": ydtTypeVerse, "стишка": ydtTypeVerse, "стишку": ydtTypeVerse, "стишком": ydtTypeVerse, "стишке": ydtTypeVerse, "стишки": ydtTypeVerse,
	"шутка": ydtTypeJoke, "шутки": ydtTypeJoke, "шутке": ydtTypeJoke, "шуткой": ydtTypeJoke, "шуток": ydtTypeJoke,
}

func yandexDialogsTalesReaction(r YandexDialogsRequest) (ydtReaction, interface{}) {
	if r.Nlu == nil && len(r.Nlu.Tokens) <= 0 {
		return ydtReactionNone, nil
	}

	var overviewState bool = false
	var randomState bool = false
	var firstNumber int = 0
	var secondNumber int = 0
	var fileType ydtFileType = ydtTypeUnknown

	var tb strings.Builder
	for _, t := range r.Nlu.Tokens {
		t = strings.ToLower(t)
		if tb.Len() > 0 {
			tb.WriteString(" ")
		}
		tb.WriteString(t)

		if ft, ok := ydtwmFileType[t]; ok {
			fileType = ft
			continue
		}

		if _, ok := ydtwmDone[t]; ok {
			return ydtReactionDone, nil
		}
		if _, ok := ydtwmNext[t]; ok {
			return ydtReactionNext, nil
		}
		if _, ok := ydtwmPrevious[t]; ok {
			return ydtReactionPrevious, nil
		}

		if _, ok := ydtwmOverview[t]; ok {
			overviewState = true
		} else if _, ok := ydtwmRandom[t]; ok {
			randomState = true
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
		return ydtReactionList, yandexDialogsTalesSlice{yandexDialogsTalesItem{fileType, firstNumber}, secondNumber}
	}
	if firstNumber > 0 {
		return ydtReactionSelect, yandexDialogsTalesSelect{yandexDialogsTalesItem{fileType, firstNumber}, true}
	}
	if overviewState {
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
			}
			if mk > 0 {
				if mk > bestMk {
					found = nil
					bestMk = mk
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
