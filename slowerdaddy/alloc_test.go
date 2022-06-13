package slowerdaddy_test

import (
	"context"
	"testing"

	"github.com/charconstpointer/slowerdaddy/slowerdaddy"
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
			a := slowerdaddy.NewDefaultAllocator(tt.fields.global, tt.fields.localLimit)
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
		name    string
		fields  fields
		args    args
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
		t.Run(tt.name, func(t *testing.T) {
			a := slowerdaddy.NewDefaultAllocator(tt.fields.global, tt.fields.localLimit)
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

//func TestAllocator_AllocConcurrent(t *testing.T) {
//	type fields struct {
//		global       *rate.Limiter
//		localLimit   int
//		globalLimit  int
//		wantOffsetBy int
//	}
//	type args struct {
//		ctx            context.Context
//		requestedQuota int
//	}
//	var tests = []struct {
//		name   string
//		args   args
//		fields fields
//	}{
//		{
//			name: "alloc concurrent",
//			args: args{
//				ctx:            context.Background(),
//				requestedQuota: 10,
//			},
//			fields: fields{
//				localLimit:   10,
//				globalLimit:  10,
//				global:       rate.NewLimiter(rate.Limit(10), 10),
//				wantOffsetBy: 1,
//			},
//		},
//		{
//			name: "alloc concurrent",
//			args: args{
//				ctx:            context.Background(),
//				requestedQuota: 4,
//			},
//			fields: fields{
//				localLimit:   4,
//				globalLimit:  4,
//				global:       rate.NewLimiter(rate.Limit(10), 10),
//				wantOffsetBy: 0,
//			},
//		},
//		{
//			name: "alloc concurrent",
//			args: args{
//				ctx:            context.Background(),
//				requestedQuota: 3,
//			},
//			fields: fields{
//				localLimit:   1,
//				globalLimit:  1,
//				global:       rate.NewLimiter(rate.Limit(10), 10),
//				wantOffsetBy: 3,
//			},
//		},
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			globalLimiter := rate.NewLimiter(rate.Limit(tt.fields.globalLimit), tt.fields.globalLimit)
//			first := slowerdaddy.NewAllocator(globalLimiter, tt.fields.localLimit)
//			second := slowerdaddy.NewAllocator(globalLimiter, tt.fields.localLimit)
//			ctx := context.Background()
//			n, err := first.Alloc(ctx, tt.args.requestedQuota)
//			if err != nil {
//				t.Errorf("first.Alloc() error = %v", err)
//			}
//			firstAllocDone := time.Now()
//			if n != tt.args.requestedQuota {
//				t.Errorf("first.Alloc() got = %v, want %v", n, tt.args.requestedQuota)
//			}
//			n, err = second.Alloc(ctx, tt.args.requestedQuota)
//			if err != nil {
//				t.Errorf("second.Alloc() error = %v", err)
//			}
//			secondAllocDone := time.Now()
//			diff := secondAllocDone.Sub(firstAllocDone).Seconds()
//			if int(diff) != tt.fields.wantOffsetBy {
//				t.Errorf("first.Alloc() and second.Alloc() should not be in the same second")
//			}
//		})
//	}
//}
