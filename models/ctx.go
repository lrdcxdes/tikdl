package models

import "context"

type DownloadContext struct {
	Context           context.Context
	MatchedContentID  string
	MatchedContentURL string
	MatchedGroups     map[string]string
	GroupSettings     *GroupSettings
	Extractor         *Extractor
	IsSpoiler         bool
}
