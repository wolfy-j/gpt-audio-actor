package listener

// Server listens for UDP audio packets and transcribes them using external tool.
// Current implement relies on a simple external CLI command, however it is possible
// to gain much higher performance via streaming approach.
