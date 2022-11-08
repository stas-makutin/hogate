package main

import (
	"bytes"
	"encoding/ascii85"
	"encoding/binary"
	"fmt"
	"io"
	"math"
)

type ydtFileType byte

const (
	ydtTypeUnknown = ydtFileType(iota)
	ydtTypeFairyTale
	ydtTypeStory
	ydtTypeSong
	ydtTypeVerse
	ydtTypeJoke
)

type ydtStateType byte

const (
	ydtStateUnknown = ydtStateType(iota)
	ydtStateItem
	ydtStateItemArray
	ydtStateSlice
	ydtStateSelect
)

type yandexDialogsTalesItem struct {
	fileType ydtFileType
	index    int32
}

func (s *yandexDialogsTalesItem) write(w io.Writer) error {
	err := binary.Write(w, binary.LittleEndian, s.fileType)
	if err == nil {
		err = binary.Write(w, binary.LittleEndian, s.index)
	}
	return err
}

func (s *yandexDialogsTalesItem) read(r io.Reader) error {
	err := binary.Read(r, binary.LittleEndian, &s.fileType)
	if err == nil {
		err = binary.Read(r, binary.LittleEndian, &s.index)
	}
	return err
}

type yandexDialogsTalesSlice struct {
	yandexDialogsTalesItem
	length int32
}

func (s *yandexDialogsTalesSlice) write(w io.Writer) error {
	err := s.yandexDialogsTalesItem.write(w)
	if err == nil {
		err = binary.Write(w, binary.LittleEndian, s.length)
	}
	return err
}

func (s *yandexDialogsTalesSlice) read(r io.Reader) error {
	err := s.yandexDialogsTalesItem.read(r)
	if err == nil {
		err = binary.Read(r, binary.LittleEndian, &s.length)
	}
	return err
}

type yandexDialogsTalesSelect struct {
	yandexDialogsTalesItem
	relative bool
}

func (s *yandexDialogsTalesSelect) write(w io.Writer) error {
	err := s.yandexDialogsTalesItem.write(w)
	if err == nil {
		err = binary.Write(w, binary.LittleEndian, s.relative)
	}
	return err
}

func (s *yandexDialogsTalesSelect) read(r io.Reader) error {
	err := s.yandexDialogsTalesItem.read(r)
	if err == nil {
		err = binary.Read(r, binary.LittleEndian, &s.relative)
	}
	return err
}

func serializeState(w io.Writer, s interface{}) (err error) {
	switch s := s.(type) {
	case yandexDialogsTalesItem:
		if err = binary.Write(w, binary.LittleEndian, ydtStateItem); err == nil {
			si := s
			err = si.write(w)
		}
	case []yandexDialogsTalesItem:
		if err = binary.Write(w, binary.LittleEndian, ydtStateItemArray); err == nil {
			items := s
			if l := len(items); l >= 0 && l <= math.MaxUint32 {
				if err = binary.Write(w, binary.LittleEndian, uint32(l)); err == nil {
					for _, v := range items {
						if err = v.write(w); err != nil {
							break
						}
					}
				}
			} else {
				err = fmt.Errorf("Number of items %d is greater than %d", l, math.MaxUint32)
			}
		}
	case yandexDialogsTalesSlice:
		if err = binary.Write(w, binary.LittleEndian, ydtStateSlice); err == nil {
			si := s
			err = si.write(w)
		}
	case yandexDialogsTalesSelect:
		if err = binary.Write(w, binary.LittleEndian, ydtStateSelect); err == nil {
			si := s
			err = si.write(w)
		}
	default:
		binary.Write(w, binary.LittleEndian, ydtStateUnknown)
	}
	return
}

func deserializeState(r io.Reader) (s interface{}, err error) {
	var t ydtStateType
	if err = binary.Read(r, binary.LittleEndian, &t); err == nil {
		switch t {
		case ydtStateItem:
			var si yandexDialogsTalesItem
			if err = si.read(r); err == nil {
				return si, nil
			}
		case ydtStateItemArray:
			var l uint32
			if err = binary.Read(r, binary.LittleEndian, &l); err == nil {
				items := make([]yandexDialogsTalesItem, l)
				i := 0
				for l > 0 {
					if err = items[i].read(r); err != nil {
						break
					}
					i++
					l--
				}
				if err == nil {
					return items, nil
				}
			}
		case ydtStateSlice:
			var si yandexDialogsTalesSlice
			if err = si.read(r); err == nil {
				return si, nil
			}
		case ydtStateSelect:
			var si yandexDialogsTalesSelect
			if err = si.read(r); err == nil {
				return si, nil
			}
		}
	}
	return nil, err
}

func encodeState(s interface{}) (string, error) {
	var b bytes.Buffer
	if err := serializeState(&b, s); err != nil {
		return "", err
	}
	if b.Len() > 0 {
		be := make([]byte, b.Len()*6)
		n := ascii85.Encode(be, b.Bytes())
		be = be[:n]
		return string(be), nil
	}
	return "", nil
}

func decodeState(s string) (interface{}, error) {
	if s == "" {
		return nil, nil
	}
	b := make([]byte, len(s))
	n, _, err := ascii85.Decode(b, []byte(s), true)
	if err != nil {
		return nil, err
	}
	b = b[:n]
	st, err := deserializeState(bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	return st, nil
}
