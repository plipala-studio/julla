package julla

type (

	Node struct {

		pattern  string

		path  string //分段路径

		match  string // regexp

		alias  string //模拟匹配别名

		nType  NodeType //路径类型

		children  Children //路径子节点

		priority  int  //优先级

		methodHandler  map[string]Handler

	}

	NodeType    string  //root static param whole regexp default

	Children  []*Node

)

