package frontend

type Variable interface{}

type API interface {
	Add(a, b Variable) Variable
	AssertIsLessOrEqual(a, b Variable)
}
