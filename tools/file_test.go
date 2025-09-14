package tools

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestFileRecords(t *testing.T) {
	t.Parallel()

	t.Run("record and retrieve file read", func(t *testing.T) {
		path := "/test/file1.txt"
		
		// Initially should have no read time
		readTime := getLastReadTime(path)
		require.True(t, readTime.IsZero())

		// Record a read
		beforeRead := time.Now()
		recordFileRead(path)
		afterRead := time.Now()

		// Should now have a read time
		readTime = getLastReadTime(path)
		require.False(t, readTime.IsZero())
		require.True(t, readTime.After(beforeRead) || readTime.Equal(beforeRead))
		require.True(t, readTime.Before(afterRead) || readTime.Equal(afterRead))
	})

	t.Run("record file write", func(t *testing.T) {
		path := "/test/file2.txt"
		
		// Record a write
		beforeWrite := time.Now()
		recordFileWrite(path)
		afterWrite := time.Now()

		// Verify write was recorded (indirectly through the map)
		fileRecordMutex.RLock()
		record, exists := fileRecords[path]
		fileRecordMutex.RUnlock()
		
		require.True(t, exists)
		require.False(t, record.writeTime.IsZero())
		require.True(t, record.writeTime.After(beforeWrite) || record.writeTime.Equal(beforeWrite))
		require.True(t, record.writeTime.Before(afterWrite) || record.writeTime.Equal(afterWrite))
	})

	t.Run("multiple operations on same file", func(t *testing.T) {
		path := "/test/file3.txt"
		
		// Record read
		recordFileRead(path)
		firstReadTime := getLastReadTime(path)
		
		// Small delay to ensure different timestamps
		time.Sleep(10 * time.Millisecond)
		
		// Record write
		recordFileWrite(path)
		
		// Read time should still be the original
		readTimeAfterWrite := getLastReadTime(path)
		require.Equal(t, firstReadTime, readTimeAfterWrite)
		
		// Record another read
		time.Sleep(10 * time.Millisecond)
		recordFileRead(path)
		secondReadTime := getLastReadTime(path)
		
		// Second read time should be different
		require.True(t, secondReadTime.After(firstReadTime))
	})

	t.Run("concurrent operations", func(t *testing.T) {
		// Test that concurrent operations don't cause race conditions
		done := make(chan bool)
		
		// Start multiple goroutines doing reads and writes
		for i := 0; i < 10; i++ {
			go func(id int) {
				path := "/test/concurrent.txt"
				for j := 0; j < 10; j++ {
					if j%2 == 0 {
						recordFileRead(path)
						getLastReadTime(path)
					} else {
						recordFileWrite(path)
					}
				}
				done <- true
			}(i)
		}
		
		// Wait for all goroutines to complete
		for i := 0; i < 10; i++ {
			<-done
		}
		
		// Verify final state is consistent
		path := "/test/concurrent.txt"
		readTime := getLastReadTime(path)
		require.False(t, readTime.IsZero())
	})
}