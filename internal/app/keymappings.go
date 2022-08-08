package app

func (a *app) loadKeyMappings() error {
	missTypeManager, err := a.loadKeysMappings()
	if err != nil {
		a.log.Errorf("loadKeyMappings missTypeManager err: %v", err)
		return err
	}
	a.missTypeManager = missTypeManager
	return nil
}
