// Code generated by mockery v1.0.0. DO NOT EDIT.

package mocks

import mock "github.com/stretchr/testify/mock"
import spotify "github.com/zmb3/spotify"

// Client is an autogenerated mock type for the Client type
type Client struct {
	mock.Mock
}

// Pause provides a mock function with given fields:
func (_m *Client) Pause() error {
	ret := _m.Called()

	var r0 error
	if rf, ok := ret.Get(0).(func() error); ok {
		r0 = rf()
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// PlayOpt provides a mock function with given fields: opt
func (_m *Client) PlayOpt(opt *spotify.PlayOptions) error {
	ret := _m.Called(opt)

	var r0 error
	if rf, ok := ret.Get(0).(func(*spotify.PlayOptions) error); ok {
		r0 = rf(opt)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// PlayerDevices provides a mock function with given fields:
func (_m *Client) PlayerDevices() ([]spotify.PlayerDevice, error) {
	ret := _m.Called()

	var r0 []spotify.PlayerDevice
	if rf, ok := ret.Get(0).(func() []spotify.PlayerDevice); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]spotify.PlayerDevice)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// TransferPlayback provides a mock function with given fields: deviceID, play
func (_m *Client) TransferPlayback(deviceID spotify.ID, play bool) error {
	ret := _m.Called(deviceID, play)

	var r0 error
	if rf, ok := ret.Get(0).(func(spotify.ID, bool) error); ok {
		r0 = rf(deviceID, play)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}