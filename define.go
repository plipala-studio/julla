package julla

var (

	Methods = [...]string{"GET", "POST", "PUT", "DELETE", "HEAD", "PATCH", "OPTIONS"}

	Equipments = [...]string{"Mobile", "Computer", "TabletPC", "TPCPlus"}
	
	//TPLViewsPath string
)

type julla struct {
	priority  int
	pattern  string
	mimicry  map[string]string
	handler  Handler
}

type H map[string]interface{} 



