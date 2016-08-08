package gearman // import "github.com/nathanaelle/gearman"

import (
	"bytes"
	"fmt"
	"testing"
)

func Test_Opaque(t *testing.T) {
	var err error

	err = opaque_test([]byte{})
	if err != nil {
		t.Errorf("got error %+v", err)
		return
	}

	err = opaque_test([]byte("hello"))
	if err != nil {
		t.Errorf("got error %+v", err)
		return
	}

}

func opaque_test(data []byte) error {
	var fn Function
	var tid TaskID

	opaq := Opacify(data)
	err := fn.Cast(opaq)
	if err != nil {
		return err
	}

	raw, err := fn.MarshalGearman()
	if err != nil {
		return err
	}
	if !bytes.Equal(data, raw) {
		return fmt.Errorf("%s MarshalGearman() expected [%v] got [%v]", "Function", data, raw)
	}

	err = tid.Cast(opaq)
	if err != nil {
		return err
	}

	raw, err = tid.MarshalGearman()
	if err != nil {
		return err
	}
	if !bytes.Equal(data, raw) {
		return fmt.Errorf("%s MarshalGearman() expected [%v] got [%v]", "TaskID", data, raw)
	}

	return nil
}