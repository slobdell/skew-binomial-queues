profile:
	go test -cpuprofile cpu.prof -memprofile mem.prof -bench .
	go tool pprof ./skewBinomialQ.test cpu.prof
