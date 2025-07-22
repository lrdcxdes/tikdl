package models

type Plugin = func(*DownloadedMedia, *DownloadConfig) error
