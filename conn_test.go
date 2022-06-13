package netlimit_test

import (
	"net"
	"testing"

	"github.com/charconstpointer/netlimit"
	"golang.org/x/time/rate"
)

func TestConn_Read(t *testing.T) {
	type fields struct {
		global      *rate.Limiter
		localLimit  int
		globalLimit int
	}
	type args struct {
		b   []byte
		msg []byte
	}
	var tests = []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "read successfully",
			fields: fields{
				localLimit:  10,
				globalLimit: 10,
				global:      rate.NewLimiter(rate.Limit(10), 10),
			},
			args: args{
				b:   make([]byte, 11),
				msg: []byte("hi there"),
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recv, sender := net.Pipe()
			defer func(recv net.Conn) {
				err := recv.Close()
				if err != nil {

				}
			}(recv)
			defer func(sender net.Conn) {
				err := sender.Close()
				if err != nil {

				}
			}(sender)
			a := netlimit.NewDefaultAllocator(tt.fields.global, tt.fields.localLimit)
			recvConn := netlimit.NewConn(recv, a)
			senderConn := netlimit.NewConn(sender, a)
			// send data to receiver
			go func() {
				_, err := senderConn.Write(tt.args.msg)
				if err != nil {

				}
			}()
			// read data from receiver
			gotN, err := recvConn.Read(tt.args.b)
			if (err != nil) != tt.wantErr {
				t.Errorf("Read() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			wantN := len(tt.args.msg)
			if gotN != wantN {
				t.Errorf("Read() gotN = %v, want %v", gotN, wantN)
			}
		})
	}
}

func TestConn_Write(t *testing.T) {
	type fields struct {
		global      *rate.Limiter
		localLimit  int
		globalLimit int
	}
	type args struct {
		b   []byte
		msg []byte
	}
	var tests = []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "write successfully",
			fields: fields{
				localLimit:  10,
				globalLimit: 10,
				global:      rate.NewLimiter(rate.Limit(10), 10),
			},
			args: args{
				b:   make([]byte, 11),
				msg: []byte("hi there"),
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recv, sender := net.Pipe()
			defer func(recv net.Conn) {
				err := recv.Close()
				if err != nil {

				}
			}(recv)
			defer func(sender net.Conn) {
				err := sender.Close()
				if err != nil {

				}
			}(sender)
			a := netlimit.NewDefaultAllocator(tt.fields.global, tt.fields.localLimit)
			recvConn := netlimit.NewConn(recv, a)
			senderConn := netlimit.NewConn(sender, a)
			// send data to receiver
			go func() {
				_, err := senderConn.Write(tt.args.msg)
				if err != nil {

				}
			}()
			// read data from receiver
			gotN, err := recvConn.Read(tt.args.b)
			if (err != nil) != tt.wantErr {
				t.Errorf("Read() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			wantN := len(tt.args.msg)
			if gotN != wantN {
				t.Errorf("Read() gotN = %v, want %v", gotN, wantN)
			}
			read := tt.args.b[:gotN]
			if string(tt.args.msg) != string(read) {
				t.Errorf("Read() gotN = %v, want %v", string(read), string(tt.args.msg))
			}
		})
	}
}
