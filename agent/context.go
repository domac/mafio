package agent

//上下文环境
type Context struct {
	Agentd *Agentd
}

func (c *Context) Logger() Logger {
	return c.Agentd.opts.Logger
}
