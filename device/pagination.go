package device

import (
	"encoding/base64"
	"encoding/json"
)

func encodePageToken(tkn RepositoryPageToken) (string, error) {
	b, err := json.Marshal(tkn)
	if err != nil {
		return "", err
	}
	enc := base64.RawURLEncoding.EncodeToString(b)
	return enc, nil
}

func decodePageToken(tkn string) (RepositoryPageToken, error) {
	dec, err := base64.RawURLEncoding.DecodeString(tkn)
	if err != nil {
		return RepositoryPageToken{}, err
	}

	var pageTkn RepositoryPageToken
	if err = json.Unmarshal(dec, &pageTkn); err != nil {
		return RepositoryPageToken{}, err
	}

	return pageTkn, nil
}
