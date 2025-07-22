package tiktok

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"regexp"

	"tikdl-web/util/networking"

	"tikdl-web/util"

	"tikdl-web/enums"

	"tikdl-web/models"

	"go.uber.org/zap"
)

const (
	appName            = "musical_ly"
	appID              = "1233"
	appVersion         = "39.8.2"
	manifestAppVersion = "2023508030"
	packageID          = "com.zhiliaoapp.musically/" + manifestAppVersion
	webBase            = "https://www.tiktok.com/@_/video/%s"
	appUserAgent       = packageID + " (Linux; U; Android 13; en_US; Pixel 7; Build/TD1A.220804.031; Cronet/58.0.2991.0)"
)

var baseHost = []string{
	"tiktok",
	"vxtiktok",
}

var VMExtractor = &models.Extractor{
	Name:       "TikTok VM",
	CodeName:   "tiktok",
	Type:       enums.ExtractorTypeSingle,
	Category:   enums.ExtractorCategorySocial,
	URLPattern: regexp.MustCompile(`https:\/\/((?:vm|vt|www)\.)?(vx)?tiktok\.com\/(?:t\/)?(?P<id>[a-zA-Z0-9-]+)`),
	Host:       baseHost,
	IsRedirect: true,
	IsHidden:   true,

	Run: func(ctx *models.DownloadContext) (*models.ExtractorResponse, error) {
		client := networking.GetExtractorHTTPClient(ctx.Extractor)
		cookies := util.GetExtractorCookies(ctx.Extractor)
		redirectURL, err := util.GetLocationURL(
			client,
			ctx.MatchedContentURL,
			webHeaders,
			cookies,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to get url location: %w", err)
		}
		parsedURL, err := url.Parse(redirectURL)
		if err != nil {
			return nil, fmt.Errorf("failed to parse redirect url: %w", err)
		}
		if parsedURL.Path == "/login" {
			zap.S().Debug("tiktok is geo restricted in your region, attemping bypass...")
			realURL := parsedURL.Query().Get("redirect_url")
			if realURL == "" {
				return nil, ErrRegionNotSupported
			}
			zap.S().Debugf("found url: %s", realURL)
			return &models.ExtractorResponse{
				URL: realURL,
			}, nil
		}
		return &models.ExtractorResponse{
			URL: redirectURL,
		}, nil
	},
}

var Extractor = &models.Extractor{
	Name:       "TikTok",
	CodeName:   "tiktok",
	Type:       enums.ExtractorTypeSingle,
	Category:   enums.ExtractorCategorySocial,
	URLPattern: regexp.MustCompile(`https?:\/\/((www|m)\.)?(vx)?tiktok\.com\/((?:embed|@[\w\.-]*)\/)?(v(ideo)?|p(hoto)?)\/(?P<id>[0-9]+)`),
	Host:       baseHost,

	Run: func(ctx *models.DownloadContext) (*models.ExtractorResponse, error) {
		// method 1: get media from API
		mediaList, err := MediaListFromAPI(ctx)
		if err == nil {
			return &models.ExtractorResponse{
				MediaList: mediaList,
			}, nil
		}
		zap.S().Debug(err)

		// method 2: get media from webpage
		mediaList, err = MediaListFromWeb(ctx)
		if err == nil {
			return &models.ExtractorResponse{
				MediaList: mediaList,
			}, nil
		}
		zap.S().Debug(err)

		return nil, ErrAllMethodsFailed
	},
}

func MediaListFromAPI(ctx *models.DownloadContext) ([]*models.Media, error) {
	var details *AwemeDetail
	var err error

	for range 10 {
		details, err = GetVideoAPI(ctx)
		if err == nil {
			break
		}
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get from api: %w", err)
	}

	media := ctx.Extractor.NewMedia(
		ctx.MatchedContentID,
		ctx.MatchedContentURL,
	)
	media.SetCaption(details.Desc)

	// Добавляем аудио формат (саундтрек)
	if details.Music != nil && details.Music.PlayURL != nil {
		media.AddFormat(&models.MediaFormat{
			FormatID:   "music_audio",
			Type:       enums.MediaTypeAudio,
			URL:        details.Music.PlayURL.URLList,
			AudioCodec: enums.MediaCodecAAC, // Предположительно AAC
			Duration:   int64(details.Music.Duration),
		})
	}

	// save to json file all response data
	var jsonFile, _ = os.Create(fmt.Sprintf("%s.json", ctx.MatchedContentID))
	defer jsonFile.Close()
	// write json raw data to file
	jsonData, _ := json.MarshalIndent(details, "", "  ")
	jsonFile.Write(jsonData)

	isImageSlide := details.ImagePostInfo != nil
	if !isImageSlide {
		video := details.Video

		// generic PlayAddr
		if video.PlayAddr != nil {
			format, err := ParsePlayAddr(video, video.PlayAddr)
			if err != nil {
				return nil, fmt.Errorf("failed to parse playaddr: %w", err)
			}
			media.AddFormat(format)
		}
		// hevc PlayAddr
		if video.PlayAddrBytevc1 != nil {
			format, err := ParsePlayAddr(video, video.PlayAddrBytevc1)
			if err != nil {
				return nil, fmt.Errorf("failed to parse playaddr: %w", err)
			}
			media.AddFormat(format)
		}
		// h264 PlayAddr
		if video.PlayAddrH264 != nil {
			format, err := ParsePlayAddr(video, video.PlayAddrH264)
			if err != nil {
				return nil, fmt.Errorf("failed to parse playaddr: %w", err)
			}
			media.AddFormat(format)
		}

		media.SetThumbnail(video.Cover.URLList)

		return []*models.Media{media}, nil
	} else {
		images := details.ImagePostInfo.Images
		mediaList := make([]*models.Media, 0, len(images))
		for i := range images {
			image := images[i]
			media.AddFormat(&models.MediaFormat{
				FormatID: "image",
				Type:     enums.MediaTypePhoto,
				URL:      image.DisplayImage.URLList,
			})
			mediaList = append(mediaList, media)
		}
		return mediaList, nil
	}
}

func MediaListFromWeb(ctx *models.DownloadContext) ([]*models.Media, error) {
	var details *WebItemStruct
	var cookies []*http.Cookie
	var err error

	// sometimes web page just returns a
	// login page, so we need to retry
	// a few times to get the correct page
	for range 5 {
		details, cookies, err = GetVideoWeb(ctx)
		if err == nil {
			break
		}
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get from web: %w", err)
	}

	media := ctx.Extractor.NewMedia(
		ctx.MatchedContentID,
		ctx.MatchedContentURL,
	)
	media.SetCaption(details.Desc)

	// Добавляем аудио формат
	if details.Music != nil && details.Music.PlayURL != nil {
		media.AddFormat(&models.MediaFormat{
			FormatID:   "web_music_audio",
			Type:       enums.MediaTypeAudio,
			URL:        details.Music.PlayURL.URLList,
			AudioCodec: enums.MediaCodecAAC,
		})
	}

	isImageSlide := details.ImagePost != nil
	if !isImageSlide {
		video := details.Video
		if video.PlayAddr != nil {
			media.AddFormat(&models.MediaFormat{
				Type:       enums.MediaTypeVideo,
				FormatID:   video.PlayAddr.URI,
				URL:        video.PlayAddr.URLList,
				VideoCodec: enums.MediaCodecAVC,
				AudioCodec: enums.MediaCodecAAC,
				Width:      video.PlayAddr.Width,
				Height:     video.PlayAddr.Height,
				Duration:   video.Duration,
				DownloadConfig: &models.DownloadConfig{
					// avoid 403 error for videos
					Cookies: cookies,
				},
				Thumbnail: details.CoverURL.URLList,
			})
			return []*models.Media{media}, nil
		}
		return nil, ErrVideoNotFound
	} else {
		images := details.ImagePost.Images
		mediaList := make([]*models.Media, 0, len(images))
		for i := range images {
			image := images[i]
			media.AddFormat(&models.MediaFormat{
				Type:      enums.MediaTypePhoto,
				FormatID:  "image",
				URL:       image.URL.URLList,
				Thumbnail: details.CoverURL.URLList,
			})
			mediaList = append(mediaList, media)
		}
		return mediaList, nil
	}
}
