package signer

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
)

const (
	licenseID      = "1611921764"
	sdkVersion     = "v05.00.06-ov-android"
	sdkVersionCode = 167775296
	platform       = "0"
)

func Sign(
	params url.Values,
	payload string,
) (map[string]string, error) {
	unix := time.Now().Unix()

	appID := params.Get("aid")
	if appID == "" {
		return nil, errors.New("missing app id")
	}
	paramsStr := params.Encode()

	// X-SS-Stub signature
	var stub string
	if payload != "" {
		hash := md5.Sum([]byte(payload))
		stub = strings.ToUpper(hex.EncodeToString(hash[:]))
	}

	// X-Gorgon signature
	gorgon := NewGorgon(
		paramsStr,
		unix,
		payload,
		"", // cookies, unused in this case
	).GetValue()

	// X-Ladon signature
	ladon, err := NewLadon(
		unix,
		licenseID,
		appID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate ladon encryption: %w", err)
	}

	// X-Argus signature
	argus, err := NewArgus(
		params,
		stub,
		unix,
		appID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate argus signature: %w", err)
	}

	headers := map[string]string{
		"X-Ss-Req-Ticket": gorgon["ticket"],
		"X-Khronos":       gorgon["khronos"],
		"X-Gorgon":        gorgon["gorgon"],
		"X-Ladon":         ladon,
		"X-Argus":         argus,
	}

	if payload != "" {
		headers["Content-length"] = strconv.Itoa(len(payload))
		headers["X-Ss-Stub"] = stub
	}

	return headers, nil
}
