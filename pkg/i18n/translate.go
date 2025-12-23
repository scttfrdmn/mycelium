package i18n

// Convenience functions that use the Global localizer

// T translates a message using the global localizer
func T(key string, data ...interface{}) string {
	if Global == nil {
		// Not initialized yet, return key
		return key
	}
	return Global.T(key, data...)
}

// Tc translates with count (for pluralization) using the global localizer
func Tc(key string, count int, data ...interface{}) string {
	if Global == nil {
		return key
	}
	return Global.Tc(key, count, data...)
}

// Tf translates with formatted data using the global localizer
func Tf(key string, data map[string]interface{}) string {
	if Global == nil {
		return key
	}
	return Global.Tf(key, data)
}

// Te translates an error message and wraps the error
func Te(key string, err error, data ...interface{}) error {
	if Global == nil {
		if err != nil {
			return err
		}
		return nil
	}
	return Global.Te(key, err, data...)
}

// MustT translates or panics (for initialization-time strings)
func MustT(key string, data ...interface{}) string {
	if Global == nil {
		panic("i18n not initialized")
	}
	return Global.MustT(key, data...)
}
