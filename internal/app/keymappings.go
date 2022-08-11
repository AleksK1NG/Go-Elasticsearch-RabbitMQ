package app

import "github.com/pkg/errors"

func (a *app) loadKeyMappings() error {
	missTypeManager, err := a.loadKeysMappings()
	if err != nil {
		a.log.Errorf("loadKeyMappings missTypeManager err: %v", err)
		return errors.Wrap(err, "loadKeysMappings")
	}
	a.missTypeManager = missTypeManager
	return nil
}
