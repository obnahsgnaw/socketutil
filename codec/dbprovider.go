package codec

type DataBuilderProvider interface {
	Provider(Name) DataBuilder
}

type Dbp struct {
	providers map[Name]DataBuilder
}

var DefaultDataBuilderProvider = NewDbp()

func NewDbp() *Dbp {
	s := &Dbp{
		providers: make(map[Name]DataBuilder),
	}
	s.Register(Json, NewJsonDataBuilder())
	s.Register(Proto, NewProtobufDataBuilder())
	return s
}

func (p *Dbp) Register(name Name, b DataBuilder) {
	p.providers[name] = b
}

func (p *Dbp) Provider(name Name) DataBuilder {
	if v, ok := p.providers[name]; ok {
		return v
	}

	return p.providers[Proto]
}
