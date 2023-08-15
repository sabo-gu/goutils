package common

import (
	"bytes"
	"encoding/gob"
)

func DeepCopy(dx interface{}, dj interface{}) error {
	var buff bytes.Buffer
	err := gob.NewEncoder(&buff).Encode(dx)
	if err != nil {
		return err
	}
	err = gob.NewDecoder(bytes.NewBuffer(buff.Bytes())).Decode(dj)
	return err
}
