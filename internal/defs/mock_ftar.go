package defs

var (
	DefFtarArray = []string{
		"/opt/flexapix/flexatar/static3/server_saves/flexatars/dmisol3c_with_image.p",
		"/opt/flexapix/flexatar/static3/server_saves/flexatars/pack0_with_image.p",
		"/opt/flexapix/flexatar/static3/server_saves/flexatars/pack2_with_image.p",
		"/opt/flexapix/flexatar/static3/server_saves/flexatars/pack1_with_image.p",
		"/opt/flexapix/flexatar/static3/server_saves/flexatars/pack3_with_image.p",
	}
	DefFtarMap = map[string]string{
		"ds":  DefFtarArray[0],
		"la1": DefFtarArray[1],
		"no1": DefFtarArray[2],
		"la2": DefFtarArray[3],
		"no2": DefFtarArray[4],
	}

	LastFtar int32
)
