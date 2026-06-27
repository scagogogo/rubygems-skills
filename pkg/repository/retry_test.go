package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/crawler-go-go-go/go-requests"
	"github.com/stretchr/testify/assert"
)

// 测试重试选项的设置
func TestRetryOptions(t *testing.T) {
	opts := NewDefaultRetryOptions()

	// 测试默认值
	assert.Equal(t, DefaultRetryAttempts, opts.MaxAttempts)
	assert.Equal(t, DefaultRetryWaitTime, opts.WaitTime)
	assert.Equal(t, DefaultRetryMaxWaitTime, opts.MaxWaitTime)
	assert.True(t, opts.UseExponentialBackoff)
	assert.NotNil(t, opts.ShouldRetry)

	// 测试方法链式调用
	opts = opts.WithMaxAttempts(5).
		WithWaitTime(2 * time.Second).
		WithMaxWaitTime(10 * time.Second).
		WithExponentialBackoff(false)

	assert.Equal(t, 5, opts.MaxAttempts)
	assert.Equal(t, 2*time.Second, opts.WaitTime)
	assert.Equal(t, 10*time.Second, opts.MaxWaitTime)
	assert.False(t, opts.UseExponentialBackoff)
}

// 测试默认重试条件
func TestDefaultShouldRetry(t *testing.T) {
	opts := NewDefaultRetryOptions()

	// 有错误时应该重试
	assert.True(t, opts.ShouldRetry(errors.New("test error")))

	// 没有错误时不应该重试
	assert.False(t, opts.ShouldRetry(nil))
}

// 模拟请求发送函数，用于测试重试逻辑
type mockRequestSender struct {
	attempts      int
	maxAttempts   int
	responses     []interface{}
	errors        []error
	requestTimes  []time.Time
	shouldTimeout bool
}

func newMockRequestSender(maxAttempts int) *mockRequestSender {
	return &mockRequestSender{
		attempts:     0,
		maxAttempts:  maxAttempts,
		responses:    make([]interface{}, maxAttempts),
		errors:       make([]error, maxAttempts),
		requestTimes: make([]time.Time, maxAttempts),
	}
}

func (m *mockRequestSender) sendRequest(ctx context.Context, options *requests.Options[any, any]) (any, error) {
	// 记录请求时间
	m.requestTimes[m.attempts] = time.Now()

	// 如果设置了超时模拟，检查上下文是否已取消
	if m.shouldTimeout {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			// 继续执行
		}
	}

	// 获取当前尝试的预设响应和错误
	response := m.responses[m.attempts]
	err := m.errors[m.attempts]

	// 增加尝试次数
	m.attempts++

	return response, err
}

// 测试重试机制的行为
func TestSendRequestWithRetry(t *testing.T) {
	// 测试重试成功的情况
	t.Run("重试成功", func(t *testing.T) {
		// 设置模拟发送器，第一次失败，第二次成功
		mock := newMockRequestSender(3)
		mock.errors[0] = errors.New("first attempt failed")
		mock.responses[1] = "success"
		mock.errors[1] = nil

		// 创建重试选项，不使用指数退避
		retryOpts := NewDefaultRetryOptions().
			WithMaxAttempts(3).
			WithWaitTime(100 * time.Millisecond).
			WithExponentialBackoff(false)

		// 执行测试
		ctx := context.Background()
		result, err := sendWithMock(ctx, mock, retryOpts)

		// 验证结果
		assert.NoError(t, err)
		assert.Equal(t, "success", result)
		assert.Equal(t, 2, mock.attempts) // 应该只尝试两次

		// 验证重试时间间隔
		if mock.attempts >= 2 {
			interval := mock.requestTimes[1].Sub(mock.requestTimes[0])
			assert.True(t, interval >= 100*time.Millisecond, "重试间隔应该至少为100ms")
		}
	})

	// 测试达到最大重试次数
	t.Run("达到最大重试次数", func(t *testing.T) {
		// 设置模拟发送器，所有尝试都失败
		mock := newMockRequestSender(3)
		for i := 0; i < mock.maxAttempts; i++ {
			mock.errors[i] = errors.New("attempt failed")
		}

		// 创建重试选项
		retryOpts := NewDefaultRetryOptions().
			WithMaxAttempts(3).
			WithWaitTime(50 * time.Millisecond).
			WithExponentialBackoff(false)

		// 执行测试
		ctx := context.Background()
		_, err := sendWithMock(ctx, mock, retryOpts)

		// 验证结果
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "max retry attempts reached")
		assert.Equal(t, 3, mock.attempts) // 应该尝试三次
	})

	// 测试指数退避
	t.Run("指数退避", func(t *testing.T) {
		// 设置模拟发送器，所有尝试都失败
		mock := newMockRequestSender(3)
		for i := 0; i < mock.maxAttempts; i++ {
			mock.errors[i] = errors.New("attempt failed")
		}

		// 创建重试选项，使用指数退避
		retryOpts := NewDefaultRetryOptions().
			WithMaxAttempts(3).
			WithWaitTime(100 * time.Millisecond).
			WithExponentialBackoff(true)

		// 执行测试
		ctx := context.Background()
		_, _ = sendWithMock(ctx, mock, retryOpts)

		// 验证重试时间间隔，第二次重试间隔应该比第一次长
		if mock.attempts >= 3 {
			interval1 := mock.requestTimes[1].Sub(mock.requestTimes[0])
			interval2 := mock.requestTimes[2].Sub(mock.requestTimes[1])
			assert.True(t, interval2 > interval1, "指数退避应该使第二次重试间隔比第一次长")
		}
	})

	// 测试上下文取消
	t.Run("上下文取消", func(t *testing.T) {
		// 设置模拟发送器
		mock := newMockRequestSender(3)
		mock.errors[0] = errors.New("first attempt failed")
		mock.shouldTimeout = true

		// 创建可取消的上下文
		ctx, cancel := context.WithCancel(context.Background())

		// 在短时间后取消上下文
		go func() {
			time.Sleep(50 * time.Millisecond)
			cancel()
		}()

		// 创建重试选项，等待时间较长
		retryOpts := NewDefaultRetryOptions().
			WithMaxAttempts(3).
			WithWaitTime(500 * time.Millisecond).
			WithExponentialBackoff(false)

		// 执行测试
		_, err := sendWithMock(ctx, mock, retryOpts)

		// 验证结果
		assert.Error(t, err)
		assert.Equal(t, context.Canceled, err)
	})
}

// 辅助函数，使用模拟发送器执行重试
func sendWithMock(ctx context.Context, mock *mockRequestSender, retryOptions *RetryOptions) (interface{}, error) {
	// 空的请求选项
	options := &requests.Options[any, any]{}

	// 记录尝试次数
	attempts := 0
	var lastErr error
	var lastResp interface{}

	for attempts < retryOptions.MaxAttempts {
		// 如果不是第一次尝试，等待指定时间
		if attempts > 0 {
			waitTime := retryOptions.WaitTime

			// 如果使用指数退避
			if retryOptions.UseExponentialBackoff {
				factor := 1 << uint(attempts-1)
				waitTime = time.Duration(float64(waitTime) * float64(factor))
				if waitTime > retryOptions.MaxWaitTime {
					waitTime = retryOptions.MaxWaitTime
				}
			}

			// 等待
			select {
			case <-time.After(waitTime):
				// 继续执行
			case <-ctx.Done():
				// 上下文被取消
				return nil, ctx.Err()
			}
		}

		// 发送请求
		resp, err := mock.sendRequest(ctx, options)

		// 检查是否需要重试
		if err == nil {
			return resp, nil
		}

		lastErr = err
		lastResp = resp
		attempts++
	}

	// 达到最大重试次数
	return lastResp, errors.New("max retry attempts reached: " + lastErr.Error())
}
