package models

import "github.com/sirupsen/logrus"

// CaptFlagsLogger logs challenge guesses in real time. It's fun to watch.
// This is mirrored from server/logger.go
var CaptFlagsLogger *logrus.Logger
