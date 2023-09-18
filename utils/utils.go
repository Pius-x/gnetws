package utils

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/samber/lo"
)

func Try(callback func() error) error {
	if r, ok := lo.TryWithErrorValue(callback); !ok {

		var err error
		if err, ok = r.(error); !ok {
			err = errors.New(fmt.Sprintf("%v", r))
		}

		return err
	}

	return nil
}
