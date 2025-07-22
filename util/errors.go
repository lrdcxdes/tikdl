package util

type Error struct {
	Message string
}

func (err *Error) Error() string {
	return err.Message
}

var (
	ErrUnavailable              = &Error{Message: "this content is unavailable"}
	ErrNotImplemented           = &Error{Message: "this feature is not implemented"}
	ErrTimeout                  = &Error{Message: "timeout error when downloading. try again"}
	ErrUnknownRIFF              = &Error{Message: "uknown RIFF format"}
	ErrUnsupportedImageFormat   = &Error{Message: "unsupported image format"}
	ErrUnsupportedExtractorType = &Error{Message: "unsupported extractor type"}
	ErrMediaGroupLimitExceeded  = &Error{Message: "media group limit exceeded for this group. try changing /settings"}
	ErrNSFWNotAllowed           = &Error{Message: "this content is marked as nsfw and can't be downloaded in this group. try changing /settings or use me privately"}
	ErrInlineMediaGroup         = &Error{Message: "you can't download media groups in inline mode. try using me in a private chat"}
	ErrAuthenticationNeeded     = &Error{Message: "this instance is not authenticated with this service."}
	ErrFileTooLarge             = &Error{Message: "file is too large for this instance"}
	ErrTelegramFileTooLarge     = &Error{Message: "file is too large for your telegram botapi. be sure to use a local botapi for large files"}
	ErrDurationTooLong          = &Error{Message: "media duration is too long for this instance"}
	ErrPaidContent              = &Error{Message: "this content is paid"}
)
