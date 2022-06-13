package netlimit_test

import (
	"context"
	"testing"

	"github.com/charconstpointer/netlimit"
	"golang.org/x/time/rate"
)

func TestAllocator_SetLimit(t *testing.T) {
	type fields struct {
		global      *rate.Limiter
		localLimit  int
		globalLimit int
	}
	type args struct {
		limit int
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "set local limit greater than global limit",
			fields: fields{
				localLimit:  0,
				globalLimit: 10,
				global:      rate.NewLimiter(rate.Limit(10), 10),
			},
			args: args{
				limit: 20,
			},
			wantErr: true,
		},
		{
			name: "set local limit successfully",
			fields: fields{
				localLimit:  0,
				globalLimit: 10,
				global:      rate.NewLimiter(rate.Limit(10), 10),
			},
			args: args{
				limit: 10,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := netlimit.NewDefaultAllocator(tt.fields.global, tt.fields.localLimit)
			if err := a.SetLimit(tt.args.limit); (err != nil) != tt.wantErr {
				t.Errorf("SetLimit() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAllocator_Alloc(t *testing.T) {
	type fields struct {
		global      *rate.Limiter
		localLimit  int
		globalLimit int
	}
	type args struct {
		ctx            context.Context
		requestedQuota int
	}
	var tests = []struct {
		args    args
		name    string
		fields  fields
		want    int
		wantErr bool
	}{
		{
			name: "alloc local limit successfully",
			fields: fields{
				localLimit:  10,
				globalLimit: 10,
				global:      rate.NewLimiter(rate.Limit(10), 10),
			},
			args: args{
				ctx:            context.Background(),
				requestedQuota: 10,
			},
			want:    10,
			wantErr: false,
		},
		{
			name: "alloc local limit greater than global limit",
			fields: fields{
				localLimit:  10,
				globalLimit: 10,
				global:      rate.NewLimiter(rate.Limit(10), 10),
			},
			args: args{
				ctx:            context.Background(),
				requestedQuota: 20,
			},
			want:    10,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		netlimit.Listen("tcp", ":0", tt.fields.globalLimit, tt.fields.localLimit)
		t.Run(tt.name, func(t *testing.T) {
			a := netlimit.NewDefaultAllocator(tt.fields.global, tt.fields.localLimit)
			got, err := a.Alloc(tt.args.ctx, tt.args.requestedQuota)
			if (err != nil) != tt.wantErr {
				t.Errorf("Alloc() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Alloc() got = %v, want %v", got, tt.want)
			}
		})
	}
}
