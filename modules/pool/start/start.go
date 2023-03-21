package main

type Config struct {
	MiningPool struct {
		Name       string `yaml:"name"`
		PoolWebUrl string `yaml:"poolweburl"`
		PoolLogDir string `yaml:"poollogdir"`
	}
}

func main() {
	// 读取配置文件

}

/*
func main() {
	// 读取配置文件
	//content, _ := ioutil.ReadFile("siaprime.yaml")

	// 1. 开启gateway模块
	i := 0
	cs, errChanCS := func() (modules.ConsensusSet, <-chan error) {
		c := make(chan error, 1)
		defer close(c)
		if params.CreateConsensusSet && params.ConsensusSet != nil {
			c <- errors.New("cannot both create consensus and use passed in consensus")
			return nil, c
		}
		if params.ConsensusSet != nil {
			return params.ConsensusSet, c
		}
		if !params.CreateConsensusSet {
			return nil, c
		}
		i++
		printfRelease("(%d/%d) Loading consensus...\n", i, numModules)
		consensusSetDeps := params.ConsensusSetDeps
		if consensusSetDeps == nil {
			consensusSetDeps = modules.ProdDependencies
		}
		return consensus.NewCustomConsensusSet(g, params.Bootstrap, filepath.Join(dir, modules.ConsensusDir), consensusSetDeps)
	}()
	if err := modules.PeekErr(errChanCS); err != nil {
		errChan <- errors.Extend(err, errors.New("unable to create consensus set"))
		return nil, errChan
	}
}
*/
