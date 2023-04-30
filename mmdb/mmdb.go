package mmdb

type MaxMindDBRecord struct {
	City struct {
		Names map[string]string `maxminddb:"names"`
	} `maxminddb:"city"`

	Continent struct {
		Code  string            `maxminddb:"code"`
		Names map[string]string `maxminddb:"names"`
	} `maxminddb:"continent"`

	Country struct {
		ISOCode string            `maxminddb:"iso_code"`
		IsEU    bool              `maxminddb:"is_in_european_union"`
		Names   map[string]string `maxminddb:"names"`
	} `maxminddb:"country"`

	Location struct {
		Latitude       float32 `maxminddb:"latitude"`
		Longitude      float32 `maxminddb:"longitude"`
		AccuracyRadius int     `maxminddb:"accuracy_radius"`
		TimeZone       string  `maxminddb:"time_zone"`
	} `maxminddb:"location"`

	Postal struct {
		Code string `maxminddb:"code"`
	} `maxminddb:"postal"`

	Network string
}

/*
	// INTERNAL STRUCTURE OF MAXMIND DB RECORDS
	map[
       city:map[geoname_id:3177363 names:map[de:Ercolano en:Ercolano fr:Ercolano pt-BR:Ercolano ru:Геркуланум]]
       continent:map[code:EU geoname_id:6255148 names:map[de:Europa en:Europe es:Europa fr:Europe ja:ヨーロッパ pt-BR:Europa ru:Европа zh-CN:欧洲]]
       country:map[geoname_id:3175395 is_in_european_union:true iso_code:IT names:map[de:Italien en:Italy es:Italia fr:Italie ja:イタリア共和国 pt-BR:Itália ru:Италия zh-CN:意大利]]
       location:map[accuracy_radius:10 latitude:40.8112 longitude:14.3528 time_zone:Europe/Rome]
       postal:map[code:80056] registered_country:map[geoname_id:3175395 is_in_european_union:true iso_code:IT names:map[de:Italien en:Italy es:Italia fr:Italie ja:イタリア共和国 pt-BR:Itália ru:Италия zh-CN:意大利]]
       subdivisions:[map[geoname_id:3181042 iso_code:72 names:map[de:Kampanien en:Campania es:Campania fr:Campanie ja:カンパニア州 pt-BR:Campânia ru:Кампания zh-CN:坎帕尼亚]] map[geoname_id:3172391 iso_code:NA names:map[de:Neapel en:Naples es:Napoles fr:Naples pt-BR:Nápoles]]]]
*/
