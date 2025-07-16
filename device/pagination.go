package device

import (
	"encoding/base64"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	iotv1 "github.com/joshjon/iot-metrics/proto/gen/iot/v1"
)

func encodePageToken(tkn RepositoryPageToken) (string, error) {
	tknpb := &iotv1.PageToken{}
	if tkn.LastID != nil {
		tknpb.OffsetId = *tkn.LastID
	}
	if tkn.LastTime != nil {
		tknpb.OffsetTime = timestamppb.New(*tkn.LastTime)
	}
	pb, err := proto.Marshal(tknpb)
	if err != nil {
		return "", err
	}

	enc := base64.RawURLEncoding.EncodeToString(pb)
	return enc, nil
}

func decodePageToken(tkn string) (RepositoryPageToken, error) {
	dec, err := base64.RawURLEncoding.DecodeString(tkn)
	if err != nil {
		return RepositoryPageToken{}, err
	}

	var tknpb iotv1.PageToken
	if err = proto.Unmarshal(dec, &tknpb); err != nil {
		return RepositoryPageToken{}, err
	}

	return RepositoryPageToken{
		LastID:   &tknpb.OffsetId,
		LastTime: ptr(tknpb.OffsetTime.AsTime()),
	}, nil
}
