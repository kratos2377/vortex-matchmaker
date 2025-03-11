goroutine 421 [running]:
log.Panicln({0xc000340e70?, 0xc000264d20?, 0xc8?})
	/snap/go/10866/src/log/log.go:446 +0x5a
github.com/kratos2377/vortex-matchmaker/domain/matchmaking.(*MatchPlayersUseCase).MatchPlayers(0xc000118140, {0x96d660, 0xc79360})
	/home/monarch/proj/vortex-matchmaker/domain/matchmaking/match_players.go:254 +0xcd8
reflect.Value.call({0x81fce0?, 0xc0001221c0?, 0x13?}, {0x8ca6f9, 0x4}, {0xc000307158, 0x1, 0x1?})
	/snap/go/10866/src/reflect/value.go:584 +0xca6
reflect.Value.Call({0x81fce0?, 0xc0001221c0?, 0xc00019cfc0?}, {0xc000307158?, 0xc000034f50?, 0xc000034540?})
	/snap/go/10866/src/reflect/value.go:368 +0xb9
github.com/go-co-op/gocron/v2.callJobFuncWithParams({0x81fce0?, 0xc0001221c0?}, {0xc0001221d0, 0x1, 0x2?})
	/home/monarch/go/pkg/mod/github.com/go-co-op/gocron/v2@v2.12.4/util.go:29 +0x1ea
github.com/go-co-op/gocron/v2.(*executor).runJob(_, {{0x96d6d0, 0xc000118190}, 0xc0001221e0, {0xb1, 0xd6, 0xf8, 0x31, 0xef, 0x42, ...}, ...}, ...)
	/home/monarch/go/pkg/mod/github.com/go-co-op/gocron/v2@v2.12.4/executor.go:403 +0x754
github.com/go-co-op/gocron/v2.(*executor).start.func1.1({{0x96d6d0, 0xc000118190}, 0xc0001221e0, {0xb1, 0xd6, 0xf8, 0x31, 0xef, 0x42, 0x4b, ...}, ...})
	/home/monarch/go/pkg/mod/github.com/go-co-op/gocron/v2@v2.12.4/executor.go:221 +0x6f
created by github.com/go-co-op/gocron/v2.(*executor).start.func1 in goroutine 420
	/home/monarch/go/pkg/mod/github.com/go-co-op/gocron/v2@v2.12.4/executor.go:220 +0x6cd
exit status 2
