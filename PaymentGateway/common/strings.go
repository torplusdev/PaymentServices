package common

import "github.com/wI2L/jettison"

func MarshalToString(obj interface{}) ([]byte, error) {

	json, err := jettison.MarshalOpts(obj,jettison.NoStringEscaping())

	return json, err
}