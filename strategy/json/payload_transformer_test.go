//go:build unit

package json_test

import (
	"encoding/json"
	anotherPayload "github.com/hellofresh/goengine/v2/internal/mocks/another/payload"
	"github.com/hellofresh/goengine/v2/internal/mocks/payload"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	strategyJSON "github.com/hellofresh/goengine/v2/strategy/json"
)

type simpleType struct {
	Test  string
	Order int
}

type anotherSimpleType struct {
	Tested  string
	Ordered int
}

type andAnotherSimpleType struct {
	Testing  string
	Ordering int
}

type unMarshableType struct {
	Bad func()
}

type order struct{ order int }
type box struct{ box int }

func TestPayloadTransformer(t *testing.T) {
	t.Run("same type on different packages", func(t *testing.T) {
		asserts := assert.New(t)

		transformer := strategyJSON.NewPayloadTransformer()
		require.NoError(t,
			transformer.RegisterPayload("payload", func() interface{} {
				return payload.Payload{}
			}),
		)

		name, data, err := transformer.ConvertPayload(anotherPayload.Payload{})
		asserts.Equal(err, strategyJSON.ErrPayloadNotRegistered)
		asserts.Equal("", name)
		asserts.Equal([]byte(nil), data)
	})
}

func TestPayloadTransformer_ConvertPayload(t *testing.T) {

	t.Run("valid tests", func(t *testing.T) {
		type testCase struct {
			title             string
			payloadInitiators map[string]strategyJSON.PayloadInitiator
			payloadType       []string
			payloadData       []interface{}
			expectedData      []string
		}

		testCases := []testCase{
			{
				"convert payload",
				map[string]strategyJSON.PayloadInitiator{
					"another": func() interface{} {
						return &anotherSimpleType{}
					},
					"tests": func() interface{} {
						return &simpleType{}
					},
					"andAnother": func() interface{} {
						return andAnotherSimpleType{}
					},
				},
				[]string{
					"tests",
					"another",
					"andAnother",
				},
				[]interface{}{
					&simpleType{Test: "test", Order: 1},
					&anotherSimpleType{Tested: "tested", Ordered: 2},
					andAnotherSimpleType{Testing: "testing", Ordering: 3},
				},
				[]string{
					`{"Test":"test","Order":1}`,
					`{"Tested":"tested","Ordered":2}`,
					`{"Testing":"testing","Ordering":3}`,
				},
			},
		}

		for _, tc := range testCases {
			t.Run(tc.title, func(t *testing.T) {
				asserts := assert.New(t)
				transformer := strategyJSON.NewPayloadTransformer()
				for payloadType, payloadInitiator := range tc.payloadInitiators {
					require.NoError(t,
						transformer.RegisterPayload(payloadType, payloadInitiator),
					)
				}

				for i, payloadData := range tc.payloadData {
					name, data, err := transformer.ConvertPayload(payloadData)
					asserts.NoError(err)
					asserts.Equal(tc.payloadType[i], name)
					asserts.JSONEq(tc.expectedData[i], string(data))
				}
			})
		}
	})

	t.Run("invalid tests", func(t *testing.T) {
		type testCase struct {
			title            string
			payloadInitiator strategyJSON.PayloadInitiator
			payloadData      interface{}
			expectedError    error
		}

		testCases := []testCase{
			{
				"not registered convert payload",
				nil,
				&simpleType{Test: "test", Order: 1},
				strategyJSON.ErrPayloadNotRegistered,
			},
			{
				"error marshalling payload",
				func() interface{} {
					// Need to register something that is not json serializable.
					return &unMarshableType{}
				},
				unMarshableType{},
				strategyJSON.ErrPayloadCannotBeSerialized,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.title, func(t *testing.T) {
				asserts := assert.New(t)
				transformer := strategyJSON.NewPayloadTransformer()
				if tc.payloadInitiator != nil {
					require.NoError(t,
						transformer.RegisterPayload("tests", tc.payloadInitiator),
					)
				}

				name, data, err := transformer.ConvertPayload(tc.payloadData)
				asserts.Equal(tc.expectedError, err)
				asserts.Equal("", name)
				asserts.Equal([]byte(nil), data)
			})
		}
	})
}

func TestJSONPayloadTransformer_CreatePayload(t *testing.T) {
	t.Run("payload creation", func(t *testing.T) {
		type validTestCase struct {
			title            string
			payloadType      string
			payloadInitiator strategyJSON.PayloadInitiator
			payloadData      interface{}
			expectedData     interface{}
		}

		testCases := []validTestCase{
			{
				"struct",
				"struct",
				func() interface{} {
					return simpleType{}
				},
				json.RawMessage(`{"test":"mine","order":1}`),
				simpleType{Test: "mine", Order: 1},
			},
			{
				"struct",
				"prt_struct",
				func() interface{} {
					return &simpleType{}
				},
				`{"test":"mine","order":1}`,
				&simpleType{Test: "mine", Order: 1},
			},
		}

		for _, testCase := range testCases {
			t.Run(testCase.title, func(t *testing.T) {
				asserts := assert.New(t)

				factory := strategyJSON.NewPayloadTransformer()
				err := factory.RegisterPayload(testCase.payloadType, testCase.payloadInitiator)
				require.NoError(t, err)

				res, err := factory.CreatePayload(testCase.payloadType, testCase.payloadData)

				asserts.EqualValues(testCase.expectedData, res)
				asserts.NoError(err)
			})
		}
	})

	t.Run("invalid arguments", func(t *testing.T) {
		type invalidTestCase struct {
			title         string
			payloadType   string
			payloadData   interface{}
			expectedError error
		}

		testCases := []invalidTestCase{
			{
				"struct payload data",
				"test",
				struct{}{},
				strategyJSON.ErrUnsupportedJSONPayloadData,
			},
			{
				"unknown payload type",
				"test",
				[]byte{},
				strategyJSON.ErrUnknownPayloadType,
			},
		}

		for _, testCase := range testCases {
			t.Run(testCase.title, func(t *testing.T) {
				factory := strategyJSON.NewPayloadTransformer()
				res, err := factory.CreatePayload(testCase.payloadType, testCase.payloadData)

				assert.Equal(t, testCase.expectedError, err)
				assert.Nil(t, res)
			})
		}
	})

	t.Run("invalid data", func(t *testing.T) {
		type invalidTestCase struct {
			title            string
			payloadInitiator strategyJSON.PayloadInitiator
			payloadData      interface{}
		}

		testCases := []invalidTestCase{
			{
				"bad json",
				func() interface{} {
					return simpleType{}
				},
				`{ bad: json }`,
			},
			{
				"bad json for a reference type",
				func() interface{} {
					return &simpleType{}
				},
				[]byte(`["comma to much",]`),
			},
		}

		for _, testCase := range testCases {
			t.Run(testCase.title, func(t *testing.T) {
				factory := strategyJSON.NewPayloadTransformer()
				require.NoError(t,
					factory.RegisterPayload("tests", testCase.payloadInitiator),
				)

				res, err := factory.CreatePayload("tests", testCase.payloadData)

				assert.IsType(t, (*json.SyntaxError)(nil), err)
				assert.Nil(t, res)
			})
		}
	})
}

func TestJSONPayloadTransformer_RegisterPayload(t *testing.T) {
	t.Run("register a type", func(t *testing.T) {
		transformer := strategyJSON.NewPayloadTransformer()
		err := transformer.RegisterPayload("test", func() interface{} {
			return &simpleType{}
		})

		assert.Nil(t, err)

		t.Run("duplicate registration", func(t *testing.T) {
			err := transformer.RegisterPayload("test", func() interface{} {
				return &simpleType{}
			})

			assert.Equal(t, strategyJSON.ErrDuplicatePayloadType, err)
		})
	})

	t.Run("failed registrations", func(t *testing.T) {
		type invalidTestCase struct {
			title            string
			payloadType      string
			payloadInitiator strategyJSON.PayloadInitiator
			expectedError    error
		}

		testCases := []invalidTestCase{
			{
				"nil initiator",
				"nil",
				func() interface{} {
					return nil
				},
				strategyJSON.ErrInitiatorInvalidResult,
			},
			{
				"nil reference initiator",
				"nil",
				func() interface{} {
					return (*invalidTestCase)(nil)
				},
				strategyJSON.ErrInitiatorInvalidResult,
			},
			{
				"anonymous initiator",
				"anonymous",
				func() interface{} {
					return &struct{ order int }{}
				},
				strategyJSON.ErrInvalidPayloadName,
			},
			{
				"empty payload type",
				"",
				func() interface{} {
					return &simpleType{}
				},
				strategyJSON.ErrInvalidPayloadType,
			},
		}

		for _, testCase := range testCases {
			t.Run(testCase.title, func(t *testing.T) {
				transformer := strategyJSON.NewPayloadTransformer()
				err := transformer.RegisterPayload(testCase.payloadType, testCase.payloadInitiator)

				assert.Equal(t, testCase.expectedError, err)
			})
		}
	})
}

func TestPayloadTransformer_RegisterMultiplePayloads(t *testing.T) {
	t.Run("register multiple types", func(t *testing.T) {
		transformer := strategyJSON.NewPayloadTransformer()
		err := transformer.RegisterPayloads(map[string]strategyJSON.PayloadInitiator{
			"order": func() interface{} {
				return &order{}
			},
			"box": func() interface{} {
				return &box{}
			},
		})

		assert.NoError(t, err)

		t.Run("duplicate registration", func(t *testing.T) {
			err := transformer.RegisterPayloads(map[string]strategyJSON.PayloadInitiator{
				"order": func() interface{} {
					return &struct{ order int }{}
				},
			})

			assert.Equal(t, strategyJSON.ErrDuplicatePayloadType, err)
		})
	})
}
