package app

func Start() error {
	conf, err := readConfig()
	if err != nil {
		return err
	}
	_, err = newGCPCollector(conf)
	if err != nil {
		return err
	}
	s := &server{
		config: conf,
	}
	return s.start()
}
