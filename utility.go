package socketio

import seri "github.com/njones/socketio/serialize"

func ampersand(s string) *string { return &s }

// stoi is string to interface
func stoi(s []string) []interface{} {
	rtn := make([]interface{}, len(s))
	for i, v := range s {
		rtn[i] = v
	}
	return rtn
}

func serviceError(err error) map[string]interface{} {
	return map[string]interface{}{"message": err.Error()}
}

func scrub(useBinary bool, event Event, data []seri.Serializable) (out interface{}, cb eventCallback, err error) {
	if !useBinary {
		rtn := make([]string, len(data)+1)
		rtn[0] = event
		for i, v := range data {
			if cbv, ok := v.(eventCallback); ok && i == len(data)-1 {
				return rtn[:len(rtn)-1], cbv, nil
			}
			rtn[i+1], err = v.Serialize()
			if err != nil {
				return nil, cb, ErrBadScrub.F(err)
			}
		}
		return rtn, nil, nil
	}
	type ifa interface{ Interface() interface{} }
	rtn := make([]interface{}, len(data)+1)
	rtn[0] = event
	for i, v := range data {
		if cbv, ok := v.(eventCallback); ok && i == len(data)-1 {
			return rtn[:len(rtn)-1], cbv, nil
		}
		if vi, ok := v.(ifa); ok {
			rtn[i+1] = vi.Interface()
			if err, ok := rtn[i+1].(error); ok {
				rtn[i+1] = err.Error()
			}
			continue
		}
		rtn[i+1] = v
	}
	return rtn, nil, nil
}
