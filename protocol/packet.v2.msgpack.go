package protocol

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io"

	"github.com/vmihailenco/msgpack"
)

func packetDataMarshalV2(idx int, r io.Reader) (out []byte, err error) {
	var raw, buf []byte
	if raw, err = io.ReadAll(r); err != nil {
		return nil, ErrReadFailed.F(err)
	}

	if len(raw) == 0 {
		return nil, nil
	}

	if buf, err = msgpack.Marshal(raw); err != nil {
		return nil, ErrEncodeFieldFailed.F(err)
	}

	return json.Marshal(struct {
		Base64 bool   `json:"base64"`
		Data   string `json:"data"`
	}{
		Base64: true,
		Data:   base64.StdEncoding.EncodeToString(buf),
	})

}

func packetDataArrayUnmarshalV2(data []byte, v interface{}) error {
	err := json.Unmarshal(data, &v)
	if err != nil {
		return ErrUnmarshalInitialFieldFailed.F(err)
	}

	switch fields := v.(type) {
	case *[]interface{}:
		if len(*fields) < 2 {
			break
		}

		for i, field := range (*fields)[1:] {
			switch datum := field.(type) {
			case map[string]interface{}:
				isBase64, _ := datum["base64"].(bool)

				if _, ok := datum["data"].(string); !ok {
					continue
				}

				var raw, buf []byte
				if isBase64 {
					if raw, err = base64.StdEncoding.DecodeString(datum["data"].(string)); err != nil {
						return ErrDecodeBase64Failed.F(err)
					}
				} else {
					rawStr, _ := datum["data"].(string)
					raw = []byte(rawStr)
				}
				if err = msgpack.Unmarshal(raw, &buf); err != nil {
					return ErrDecodeFieldFailed.F(err)
				}

				(*fields)[i+1] = bytes.NewReader(buf) // +1 because we started 1 ahead...
			}
		}
	}
	return nil
}

func packetDataObjectUnmarshalV2(data []byte, v interface{}) error {
	err := json.Unmarshal(data, &v)
	if err != nil {
		return ErrUnmarshalInitialFieldFailed.F(err)
	}

	switch fields := v.(type) {
	case *map[string]interface{}:

		for i, field := range *fields {
			switch datum := field.(type) {
			case map[string]interface{}:
				if isBase64, ok := datum["base64"].(bool); !ok || !isBase64 {
					continue
				} else if _, ok := datum["data"].(string); !ok {
					continue
				}

				var raw, buf []byte
				if raw, err = base64.StdEncoding.DecodeString(datum["data"].(string)); err != nil {
					return ErrDecodeBase64Failed.F(err)
				}
				if err = msgpack.Unmarshal(raw, &buf); err != nil {
					return ErrDecodeFieldFailed.F(err)
				}

				(*fields)[i] = bytes.NewReader(buf)
			}
		}
	}

	return nil
}

// Base64 encoding support. The example is in the commit message:
// https://github.com/socketio/socket.io-protocol/commit/a60fd25ee949d98f592848b994d2bfbacc804564

func (pac *PacketV2) WithData(x interface{}) Packet {
	pac.packet.WithData(x)
	switch pac.Data.(type) {
	case *packetDataArray:
		pac.Data.(*packetDataArray).marshalBinary = packetDataMarshalV2
		pac.Data.(*packetDataArray).unmarshalBinary = packetDataArrayUnmarshalV2
	case *packetDataObject:
		pac.Data.(*packetDataObject).marshalBinary = packetDataMarshalV2
		pac.Data.(*packetDataObject).unmarshalBinary = packetDataObjectUnmarshalV2
	}
	return pac
}
