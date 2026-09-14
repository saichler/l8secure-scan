/*
 * © 2025 Sharon Aicler (saichler@gmail.com)
 *
 * Layer 8 Ecosystem is licensed under the Apache License, Version 2.0.
 * You may obtain a copy of the License at:
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package logs

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"sync/atomic"
	"time"

	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/l8utils/go/utils/queues"
)

// TailFile continuously monitors a log file and transmits new lines to the log server.
// It handles file truncation by detecting when the file size becomes smaller.
// The location parameter specifies where to start tailing:
//   - location == 0: start from the beginning of the file
//   - location == -1: start from the end of the file (only new lines)
//   - location > 0: start from the specific byte offset (useful for resuming after crash)
func TailFile(filename string, nic ifs.IVNic, location int64) error {
	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Get initial file size before seeking
	fileInfo, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat file: %w", err)
	}
	initialSize := fileInfo.Size()

	// Seek to the appropriate position based on location parameter
	var lastSize int64
	if location == -1 {
		// Start from the end of the file (only new lines)
		_, err = file.Seek(0, io.SeekEnd)
		if err != nil {
			return fmt.Errorf("failed to seek to end: %w", err)
		}
		lastSize = initialSize
	} else if location == 0 {
		// Start from the beginning of the file
		_, err = file.Seek(0, io.SeekStart)
		if err != nil {
			return fmt.Errorf("failed to seek to start: %w", err)
		}
		lastSize = 0
	} else {
		// Start from a specific byte offset (resume after crash)
		_, err = file.Seek(location, io.SeekStart)
		if err != nil {
			return fmt.Errorf("failed to seek to position %d: %w", location, err)
		}
		lastSize = location
	}

	reader := bufio.NewReader(file)

	// Buffer configuration tuned for ~100 concurrent tailers
	const pollInterval = 1 * time.Second
	const cooldownDuration = 2 * time.Second
	const maxBufferAge = 30 * time.Second
	const maxBufferBytes = 512 * 1024 // 512KB

	lineBuffer := queues.NewQueue("LineBuffer", 30000)
	bufferBytes := 0
	lastLineTime := time.Now()
	var lastFlushTime atomic.Int64
	lastFlushTime.Store(time.Now().UnixNano())

	flushBuffer := func() {
		lines := lineBuffer.Clear()
		if len(lines) > 0 {
			localBuff := make([]string, len(lines))
			for i, line := range lines {
				localBuff[i] = line.(string)
			}
			SendLogs(filename, nic, localBuff...)
			lastFlushTime.Store(time.Now().UnixNano())
		}
	}

	for {
		// Check current file size — single Stat per iteration
		fileInfo, err := file.Stat()
		if err != nil {
			return fmt.Errorf("failed to stat file: %w", err)
		}
		currentSize := fileInfo.Size()

		// Detect truncation: if current size is less than last known size
		if currentSize < lastSize {
			go flushBuffer()
			bufferBytes = 0

			file.Close()
			file, err = os.Open(filename)
			if err != nil {
				return fmt.Errorf("failed to reopen file after truncation: %w", err)
			}

			reader = bufio.NewReader(file)
			lastSize = 0
			currentSize = 0
		}

		// Skip reading if file hasn't grown — avoids ReadString calls on idle files
		hasNewLines := false
		if currentSize > lastSize {
			for {
				line, err := reader.ReadString('\n')
				if err != nil {
					if err == io.EOF {
						break
					}
					return fmt.Errorf("error reading file: %w", err)
				}

				lineBuffer.Add(line)
				bufferBytes += len(line)
				lastLineTime = time.Now()
				hasNewLines = true

				// Flush if buffer reaches max bytes to cap memory
				if bufferBytes >= maxBufferBytes {
					go flushBuffer()
					bufferBytes = 0
				}
			}
			lastSize = currentSize
		}

		// Check flush conditions
		timeSinceLastLine := time.Since(lastLineTime)
		lastFlushTimeValue := time.Unix(0, lastFlushTime.Load())
		timeSinceLastFlush := time.Since(lastFlushTimeValue)

		if lineBuffer.Size() > 0 {
			if (!hasNewLines && timeSinceLastLine >= cooldownDuration) ||
				timeSinceLastFlush >= maxBufferAge {
				go flushBuffer()
				bufferBytes = 0
			}
		}

		time.Sleep(pollInterval)
	}
}
