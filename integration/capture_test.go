package integration

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/jintilm/GrantDiff/internal/cluster"
	"github.com/jintilm/GrantDiff/internal/model"
	rbac "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	kubetesting "k8s.io/client-go/testing"
)

// Controllers may finish creating or deleting identity objects between reads.
// Retry only the explicit consistency error, before authorization assertions run.
func captureStable(t *testing.T, ctx context.Context, client kubernetes.Interface) (*model.State, cluster.Snapshot, []byte, error) {
	t.Helper()
	for attempt := 1; ; attempt++ {
		if err := ctx.Err(); err != nil {
			return nil, cluster.Snapshot{}, nil, err
		}
		state, metadata, data, err := cluster.Capture(ctx, client)
		if !errors.Is(err, cluster.ErrCaptureChanged) || attempt == 5 {
			return state, metadata, data, err
		}
		t.Logf("snapshot changed during collection, retrying after attempt %d of 5", attempt)
		select {
		case <-ctx.Done():
			return nil, metadata, nil, ctx.Err()
		case <-time.After(100 * time.Millisecond):
		}
	}
}

func TestCaptureStableHandlesOnlyObservedDrift(t *testing.T) {
	apiError := errors.New("list request failed")
	for _, test := range []struct {
		name       string
		changing   bool
		apiFailure bool
		canceled   bool
		wantReads  int
		wantError  error
	}{
		{name: "settles after one capture", wantReads: 4},
		{name: "continuing drift stops after five captures", changing: true, wantReads: 10, wantError: cluster.ErrCaptureChanged},
		{name: "API failure returns immediately", apiFailure: true, wantReads: 1, wantError: apiError},
		{name: "cancellation returns immediately", canceled: true, wantReads: 0, wantError: context.Canceled},
	} {
		t.Run(test.name, func(t *testing.T) {
			client := fake.NewClientset()
			reads := 0
			client.PrependReactor("list", "roles", func(kubetesting.Action) (bool, runtime.Object, error) {
				reads++
				if test.apiFailure {
					return true, nil, apiError
				}
				name := "stable"
				if test.changing || reads == 1 {
					name = fmt.Sprintf("changing-%d", reads)
				}
				return true, &rbac.RoleList{Items: []rbac.Role{{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "test"}, Rules: []rbac.PolicyRule{{APIGroups: []string{""}, Resources: []string{"configmaps"}, Verbs: []string{"get"}}}}}}, nil
			})
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if test.canceled {
				cancel()
			}
			state, metadata, _, err := captureStable(t, ctx, client)
			if !errors.Is(err, test.wantError) || reads != test.wantReads {
				t.Fatalf("reads=%d err=%v, want reads=%d err=%v", reads, err, test.wantReads, test.wantError)
			}
			if err == nil && (state.Objects["Role/test/stable"] == nil || metadata.Consistency != "two-equal-observations") {
				t.Fatal("capture accepted an unstable state")
			}
		})
	}
}
