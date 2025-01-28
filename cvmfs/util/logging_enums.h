/**
 * This file is part of the CernVM File System.
 */

#ifndef CVMFS_UTIL_LOGGING_ENUMS_H_
#define CVMFS_UTIL_LOGGING_ENUMS_H_

enum LogFacilities {
  kLogDebug = 0x01,
  kLogStdout = 0x02,
  kLogStderr = 0x04,
  kLogSyslog = 0x08,
  kLogSyslogWarn = 0x10,
  kLogSyslogErr = 0x20,
  kLogCustom0 = 0x40,
  kLogCustom1 = 0x80,
  kLogCustom2 = 0x100,
};

/**
 * Changes in this enum must be done in logging.cc as well!
 * (see const char *module_names[] = {....})
 */
enum LogSource {
  kLogCache = 1,
  kLogCatalog,
  kLogSql,
  kLogCvmfs,
  kLogHash,
  kLogDownload,
  kLogCompress,
  kLogQuota,
  kLogTalk,
  kLogMonitor,
  kLogLru,
  kLogFuse,
  kLogSignature,
  kLogFsTraversal,
  kLogCatalogTraversal,
  kLogNfsMaps,
  kLogPublish,
  kLogSpooler,
  kLogConcurrency,
  kLogUtility,
  kLogGlueBuffer,
  kLogHistory,
  kLogUnionFs,
  kLogPathspec,
  kLogReceiver,
  kLogUploadS3,
  kLogUploadGateway,
  kLogS3Fanout,
  kLogGc,
  kLogDns,
  kLogAuthz,
  kLogReflog,
  kLogKvStore,
  kLogTelemetry,
  kLogCurl
};

enum LogFlags {
  kLogNoLinebreak = 0x200,
  kLogShowSource  = 0x400,
  kLogSensitive   = 0x800,  ///< Don't add the line to the memory log buffer
};

enum LogLevels {
  kLogLevel0   = 0x01000,
  kLogNormal   = 0x02000,
  kLogInform   = 0x04000,
  kLogVerbose  = 0x08000,
  kLogNone     = 0x10000,
};

const int kLogWarning = kLogStdout | kLogShowSource | kLogNormal;
const int kLogInfoMsg = kLogStdout | kLogShowSource | kLogInform;
const int kLogVerboseMsg = kLogStdout | kLogShowSource | kLogVerbose;

#endif // CVMFS_UTIL_LOGGING_ENUMS_H_