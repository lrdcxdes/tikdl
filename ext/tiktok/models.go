package tiktok

type Response struct {
	AwemeDetails []*AwemeDetail `json:"aweme_details"`
	StatusCode   int            `json:"status_code"`
	StatusMsg    string         `json:"status_msg"`
}

type Cover struct {
	Height    int64    `json:"height"`
	URI       string   `json:"uri"`
	URLList   []string `json:"url_list"`
	URLPrefix any      `json:"url_prefix"`
	Width     int64    `json:"width"`
}

type PlayAddr struct {
	DataSize int64    `json:"data_size"`
	FileCs   string   `json:"file_cs"`
	FileHash string   `json:"file_hash"`
	Height   int64    `json:"height"`
	URI      string   `json:"uri"`
	URLKey   string   `json:"url_key"`
	URLList  []string `json:"url_list"`
	Width    int64    `json:"width"`
}

type Image struct {
	DisplayImage *DisplayImage `json:"display_image"`
}

type DisplayImage struct {
	Height    int      `json:"height"`
	URI       string   `json:"uri"`
	URLList   []string `json:"url_list"`
	URLPrefix any      `json:"url_prefix"`
	Width     int      `json:"width"`
}

type ImagePostInfo struct {
	Images      []Image `json:"images"`
	MusicVolume float64 `json:"music_volume"`
	PostExtra   string  `json:"post_extra"`
	Title       string  `json:"title"`
}

type Video struct {
	Cover           Cover     `json:"cover"`
	Duration        int64     `json:"duration"`
	HasWatermark    bool      `json:"has_watermark"`
	Height          int64     `json:"height"`
	PlayAddr        *PlayAddr `json:"play_addr"`
	PlayAddrBytevc1 *PlayAddr `json:"play_addr_bytevc1"`
	PlayAddrH264    *PlayAddr `json:"play_addr_h264"`
	Width           int64     `json:"width"`
}

// --- Добавьте эти структуры ---
type MusicPlayURL struct {
	URI     string   `json:"uri"`
	URLList []string `json:"url_list"`
}

type MusicInfo struct {
	ID       int64         `json:"id"`
	PlayURL  *MusicPlayURL `json:"play_url"`
	Duration int           `json:"duration"`
	Title    string        `json:"title"`
	Author   string        `json:"author"`
}

// -----------------------------

type AwemeDetail struct {
	AwemeID       string         `json:"aweme_id"`
	AwemeType     int            `json:"aweme_type"`
	Desc          string         `json:"desc"`
	Video         *Video         `json:"video"`
	ImagePostInfo *ImagePostInfo `json:"image_post_info"`
	Music         *MusicInfo     `json:"music"` // <--- ДОБАВИТЬ ЭТО ПОЛЕ
}

type WebItemStruct struct {
	ID        string        `json:"id"`
	Desc      string        `json:"desc"`
	Video     *WebVideo     `json:"video"`
	ImagePost *WebImagePost `json:"imagePost"`
	Music     *MusicInfo    `json:"music"`      // <--- ДОБАВИТЬ ЭТО ПОЛЕ
	CoverURL  *WebImageURL  `json:"videoCover"` // <--- ДОБАВИТЬ ЭТО ПОЛЕ ДЛЯ ОБЛОЖКИ
}

type WebImagePost struct {
	Images []*WebImage `json:"images"`
	Title  string      `json:"title"`
}

type WebVideo struct {
	Duration int64        `json:"duration"`
	Height   int64        `json:"height"`
	PlayAddr *WebPlayAddr `json:"PlayAddrStruct"`
	Width    int64        `json:"width"`
}

type WebImageURL struct {
	URLList []string `json:"urlList"`
}

type WebImage struct {
	URL *WebImageURL `json:"imageURL"`
}

type WebPlayAddr struct {
	FileHash string   `json:"FileHash"`
	FileCs   string   `json:"FileCs"`
	DataSize string   `json:"DataSize"`
	Width    int64    `json:"Width"`
	Height   int64    `json:"Height"`
	URI      string   `json:"Uri"`
	URLList  []string `json:"UrlList"`
	URLKey   string   `json:"UrlKey"`
}
