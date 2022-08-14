package app

import "github.com/pkg/errors"

func (a *app) loadKeyMappings() error {
	keyboardLayoutManager, err := a.loadKeyboardLayoutManager()
	if err != nil {
		a.log.Errorf("loadKeyMappings keyboardLayoutManager err: %v", err)
		return errors.Wrap(err, "loadKeyboardLayoutManager")
	}
	a.keyboardLayoutManager = keyboardLayoutManager
	return nil
}
