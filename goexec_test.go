package goexec

import (
	"bytes"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestCmd(t *testing.T) {
	t.Run("multiple starts", func(t *testing.T) {
		cmd := Command("sleep", "1")

		err := cmd.Start()
		require.NoError(t, err)

		err = cmd.Start()
		require.NoError(t, err)
	})

	t.Run("start after exit", func(t *testing.T) {
		cmd := Command("echo", "ok")

		err := cmd.Run()
		require.NoError(t, err)

		err = cmd.Start()
		require.Error(t, err)
	})

	t.Run("multiple waits", func(t *testing.T) {
		cmd := Command("echo", "ok")

		err := cmd.Start()
		require.NoError(t, err)

		err = cmd.Wait()
		require.NoError(t, err)

		err = cmd.Wait()
		require.NoError(t, err)
	})

	t.Run("exit code", func(t *testing.T) {
		cmd := Command("echo", "ok")

		err := cmd.Run()
		require.NoError(t, err)
		require.True(t, cmd.Exited())

		code, err := cmd.ExitCode()
		require.NoError(t, err)
		require.Equal(t, 0, code)
	})

	t.Run("with output", func(t *testing.T) {
		b := &bytes.Buffer{}
		cmd := Command("echo", "ok").WithOutput(b)

		err := cmd.Run()
		require.NoError(t, err)
		require.Equal(t, "ok\n", b.String())
	})

	t.Run("with input", func(t *testing.T) {
		in := &bytes.Buffer{}
		in.Write([]byte("1 2\n"))

		out := &bytes.Buffer{}
		cmd := Command("cut", "-d", " ", "-f", "2").WithInput(in).WithOutput(out)

		err := cmd.Run()
		require.NoError(t, err)

		time.Sleep(10 * time.Millisecond)
		require.Equal(t, "2\n", out.String())
	})

	t.Run("many waits concurrent", func(t *testing.T) {
		cmd := Command("echo", "ok")
		err := cmd.Start()
		require.NoError(t, err)

		wg := sync.WaitGroup{}
		wg.Add(100)
		for i := 0; i < 100; i++ {
			go func() {
				defer wg.Done()
				err := cmd.Wait()
				require.NoError(t, err)
			}()
		}
	})
}
