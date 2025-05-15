/**
 * This file is part of the CernVM File System.
 */


#include "publish/repository.h"

#include <fcntl.h>
#include <unistd.h>

#include <cassert>
#include <string>

#include "backoff.h"
#include "catalog_mgr_ro.h"
#include "crypto/hash.h"
#include "directory_entry.h"
#include "duplex_curl.h"
#include "gateway_util.h"
#include "json_document.h"
#include "publish/except.h"
#include "ssl.h"
#include "upload.h"
#include "util/logging.h"
#include "util/pointer.h"
#include "util/posix.h"
#include "util/string.h"

namespace {

struct CurlBuffer {
  std::string data;
};

static CURL* PrepareCurl(const std::string& method) {
  const char* user_agent_string = "cvmfs/" CVMFS_VERSION;

  CURL* h_curl = curl_easy_init();
  assert(h_curl != NULL);

  curl_easy_setopt(h_curl, CURLOPT_NOPROGRESS, 1L);
  curl_easy_setopt(h_curl, CURLOPT_USERAGENT, user_agent_string);
  curl_easy_setopt(h_curl, CURLOPT_MAXREDIRS, 50L);
  curl_easy_setopt(h_curl, CURLOPT_CUSTOMREQUEST, method.c_str());

  return h_curl;
}

static size_t RecvCB(void* buffer, size_t size, size_t nmemb, void* userp) {
  CurlBuffer* my_buffer = static_cast<CurlBuffer*>(userp);

  if (size * nmemb < 1) {
    return 0;
  }

  my_buffer->data = static_cast<char*>(buffer);

  return my_buffer->data.size();
}

/**
 * @return true if request failed because of unreachable gateway, false otherwise
 */
static bool MakeAcquireRequest(
  const gateway::GatewayKey &key,
  const std::string& repo_path,
  const std::string& repo_service_url,
  int llvl,
  CurlBuffer* buffer)
{
  CURLcode ret = static_cast<CURLcode>(0);

  CURL* h_curl = PrepareCurl("POST");

  const std::string payload = "{\"path\" : \"" + repo_path +
                              "\", \"api_version\" : \"" +
                              StringifyInt(gateway::APIVersion()) + "\", " +
                              "\"hostname\" : \"" + GetHostname() + "\"}";

  shash::Any hmac(shash::kSha1);
  shash::HmacString(key.secret(), payload, &hmac);
  SslCertificateStore cs;
  cs.UseSystemCertificatePath();
  cs.ApplySslCertificatePath(h_curl);

  const std::string header_str =
    std::string("Authorization: ") + key.id() + " " +
    Base64(hmac.ToString(false));
  struct curl_slist* auth_header = NULL;
  auth_header = curl_slist_append(auth_header, header_str.c_str());
  curl_easy_setopt(h_curl, CURLOPT_HTTPHEADER, auth_header);

  // Make request to acquire lease from repo services
  curl_easy_setopt(h_curl, CURLOPT_URL, (repo_service_url + "/leases").c_str());
  curl_easy_setopt(h_curl, CURLOPT_POSTFIELDSIZE_LARGE,
                   static_cast<curl_off_t>(payload.length()));
  curl_easy_setopt(h_curl, CURLOPT_POSTFIELDS, payload.c_str());
  curl_easy_setopt(h_curl, CURLOPT_WRITEFUNCTION, RecvCB);
  curl_easy_setopt(h_curl, CURLOPT_WRITEDATA, buffer);

  ret = curl_easy_perform(h_curl);
  curl_easy_cleanup(h_curl);
  if (ret == CURLE_COULDNT_CONNECT) {
    LogCvmfs(kLogUploadGateway, llvl | kLogStderr,
             "Make lease acquire request failed: %d. Reply: %s", ret,
             buffer->data.c_str());
    return true;
  }
  if (ret != CURLE_OK) {
    LogCvmfs(kLogUploadGateway, llvl | kLogStderr,
             "Make lease acquire request failed: %d. Reply: %s", ret,
             buffer->data.c_str());
    throw publish::EPublish("cannot acquire lease",
                            publish::EPublish::kFailLeaseHttp);
  }
  return false;
}

// TODO(jblomer): This should eventually also handle the POST request for
// committing a transaction
/**
 * @return true if request failed because of unreachable gateway, false otherwise
 */
static bool MakeDropRequest(
  const gateway::GatewayKey &key,
  const std::string &session_token,
  const std::string &repo_service_url,
  int llvl,
  CurlBuffer *reply)
{
  CURLcode ret = static_cast<CURLcode>(0);

  CURL *h_curl = PrepareCurl("DELETE");

  shash::Any hmac(shash::kSha1);
  shash::HmacString(key.secret(), session_token, &hmac);
  SslCertificateStore cs;
  cs.UseSystemCertificatePath();
  cs.ApplySslCertificatePath(h_curl);

  const std::string header_str =
    std::string("Authorization: ") + key.id() + " " +
    Base64(hmac.ToString(false));
  struct curl_slist *auth_header = NULL;
  auth_header = curl_slist_append(auth_header, header_str.c_str());
  curl_easy_setopt(h_curl, CURLOPT_HTTPHEADER, auth_header);

  curl_easy_setopt(h_curl, CURLOPT_URL,
                   (repo_service_url + "/leases/" + session_token).c_str());
  curl_easy_setopt(h_curl, CURLOPT_POSTFIELDSIZE_LARGE,
                   static_cast<curl_off_t>(0));
  curl_easy_setopt(h_curl, CURLOPT_POSTFIELDS, NULL);
  curl_easy_setopt(h_curl, CURLOPT_WRITEFUNCTION, RecvCB);
  curl_easy_setopt(h_curl, CURLOPT_WRITEDATA, reply);

  ret = curl_easy_perform(h_curl);
  curl_easy_cleanup(h_curl);
  if (ret == CURLE_COULDNT_CONNECT) {
    LogCvmfs(kLogUploadGateway, llvl | kLogStderr,
             "Make lease drop request failed: %d. Reply: '%s'. Gateway most likely unavailable.",
             ret, reply->data.c_str());
    return true;
  }
  if (ret != CURLE_OK) {
    LogCvmfs(kLogUploadGateway, llvl | kLogStderr,
             "Make lease drop request failed: %d. Reply: '%s'",
             ret, reply->data.c_str());
    throw publish::EPublish("cannot drop lease",
                            publish::EPublish::kFailLeaseHttp);
  }

  return false;
}

static LeaseReply ParseAcquireReply(
  const CurlBuffer &buffer,
  std::string *session_token,
  int llvl)
{
  if (buffer.data.size() == 0 || session_token == NULL) {
    return kLeaseReplyFailure;
  }

  const UniquePtr<JsonDocument> reply(JsonDocument::Create(buffer.data));
  if (!reply.IsValid() || !reply->IsValid()) {
    return kLeaseReplyFailure;
  }

  const JSON *result =
      JsonDocument::SearchInObject(reply->root(), "status", JSON_STRING);
  if (result != NULL) {
    const std::string status = result->string_value;
    if (status == "ok") {
      LogCvmfs(kLogCvmfs, llvl | kLogStdout, "Gateway reply: ok");
      const JSON *token = JsonDocument::SearchInObject(
          reply->root(), "session_token", JSON_STRING);
      if (token != NULL) {
        LogCvmfs(kLogCvmfs, kLogDebug, "Session token: %s",
                 token->string_value);
        *session_token = token->string_value;
        return kLeaseReplySuccess;
      }
    } else if (status == "path_busy") {
      const JSON *time_remaining = JsonDocument::SearchInObject(
          reply->root(), "time_remaining", JSON_STRING);
      LogCvmfs(kLogCvmfs, llvl | kLogStdout,
               "Path busy. Time remaining = %s", (time_remaining != NULL) ?
               time_remaining->string_value : "UNKNOWN");
      return kLeaseReplyBusy;
    } else if (status == "error") {
      const JSON *reason =
          JsonDocument::SearchInObject(reply->root(), "reason", JSON_STRING);
      LogCvmfs(kLogCvmfs, llvl | kLogStdout, "Error: '%s'",
               (reason != NULL) ? reason->string_value : "");
    } else {
      LogCvmfs(kLogCvmfs, llvl | kLogStdout, "Unknown reply. Status: %s",
               status.c_str());
    }
  }

  return kLeaseReplyFailure;
}


static LeaseReply ParseDropReply(const CurlBuffer &buffer, int llvl) {
  if (buffer.data.size() == 0) {
    return kLeaseReplyFailure;
  }

  const UniquePtr<const JsonDocument> reply(JsonDocument::Create(buffer.data));
  if (!reply.IsValid() || !reply->IsValid()) {
    return kLeaseReplyFailure;
  }

  const JSON *result =
      JsonDocument::SearchInObject(reply->root(), "status", JSON_STRING);
  if (result != NULL) {
    const std::string status = result->string_value;
    if (status == "ok") {
      LogCvmfs(kLogCvmfs, llvl | kLogStdout, "Gateway reply: ok");
      return kLeaseReplySuccess;
    } else if (status == "invalid_token") {
      LogCvmfs(kLogCvmfs, llvl | kLogStdout, "Error: invalid session token");
    } else if (status == "error") {
      const JSON *reason =
          JsonDocument::SearchInObject(reply->root(), "reason", JSON_STRING);
      LogCvmfs(kLogCvmfs, llvl | kLogStdout, "Error from gateway: '%s'",
               (reason != NULL) ? reason->string_value : "");
    } else {
      LogCvmfs(kLogCvmfs, llvl | kLogStdout, "Unknown reply. Status: %s",
               status.c_str());
    }
  }

  return kLeaseReplyFailure;
}

}  // anonymous namespace

namespace publish {

Publisher::Session::Session(const Settings &settings_session)
  : settings_(settings_session)
  , keep_alive_(false)
  // TODO(jblomer): it would be better to actually read & validate the token
  , has_lease_(FileExists(settings_.token_path))
{
}


Publisher::Session::Session(const SettingsPublisher &settings_publisher,
                            int llvl)
{
  keep_alive_ = false;
  if (settings_publisher.storage().type() != upload::SpoolerDefinition::Gateway)
  {
    has_lease_ = true;
    return;
  }

  settings_.service_endpoint = settings_publisher.storage().endpoint();
  settings_.repo_path = settings_publisher.fqrn() + "/" +
                        settings_publisher.transaction().lease_path();
  settings_.gw_key_path = settings_publisher.keychain().gw_key_path();
  settings_.token_path =
    settings_publisher.transaction().spool_area().gw_session_token();
  settings_.llvl = llvl;

  // TODO(jblomer): it would be better to actually read & validate the token
  has_lease_ = FileExists(settings_.token_path);
  // If a lease is already present, we don't want to remove it automatically
  keep_alive_ = has_lease_;
}


void Publisher::Session::SetKeepAlive(bool value) {
  keep_alive_ = value;
}

/**
 * Update the local token ring SQLite database, by retreiving the information from the best available gateway
 * @param repo_path The path to the repository
 * @return true if request failed because of unreachable gateway, false otherwise
 */
bool Publisher::Session::UpdateGatewayDb(const std::string& repo_path, const std::string& src_gateway) const {
  // Extract the repo name from the repo path
  size_t end_pos = repo_path.find('/', 0);
  if (end_pos == 0) {
    throw EPublish("repo_path cannot start with a '/'", EPublish::kFailInput);
  }
  if (end_pos == std::string::npos) {
    throw EPublish("repo_path must contain a '/'", EPublish::kFailInput);
  }

  std::string repo_name = repo_path.substr(0, end_pos);

  // Make curl call to gateway to retrieve gateway addresses
  CurlBuffer buffer;
  CURL* h_curl = PrepareCurl("GET");
  std::string url = src_gateway + "/token-ring";

  const std::string payload = "{\"repo\" : \"" + repo_name + "\"}";
  curl_easy_setopt(h_curl, CURLOPT_POSTFIELDSIZE_LARGE,
           static_cast<curl_off_t>(payload.length()));
  curl_easy_setopt(h_curl, CURLOPT_POSTFIELDS, payload.c_str());
  curl_easy_setopt(h_curl, CURLOPT_URL, url.c_str());
  curl_easy_setopt(h_curl, CURLOPT_WRITEFUNCTION, RecvCB);
  curl_easy_setopt(h_curl, CURLOPT_WRITEDATA, &buffer);

  CURLcode ret = curl_easy_perform(h_curl);
  curl_easy_cleanup(h_curl);

  if (ret == CURLE_COULDNT_CONNECT) {
    LogCvmfs(kLogUploadGateway, settings_.llvl | kLogStderr,
             "failed to retrieve gateway addresses: %d. Reply: %s. Gateway most likely not available.", ret,
             buffer.data.c_str());
    return true;
  }
  if (ret != CURLE_OK) {
    throw EPublish("failed to retrieve gateway addresses: " + std::string(curl_easy_strerror(ret)),
                   EPublish::kFailGatewayKey);
  }

  // Interpret json
  const UniquePtr<JsonDocument> reply(JsonDocument::Create(buffer.data));
  if (!reply.IsValid() || !reply->IsValid()) {
    throw EPublish("failed to parse gateway addresses", EPublish::kFailGatewayKey);
  }

  const JSON* gateways_array = JsonDocument::SearchInObject(reply->root(), "gateways", JSON_ARRAY);
  if (gateways_array == NULL) {
    throw EPublish("no 'gateways' array found in the response", EPublish::kFailGatewayKey);
  }
  const JSON* gateway = gateways_array->first_child;

  sqlite3* db = NULL;
  std::string db_path = "/var/spool/cvmfs/" + repo_name + "/tokenring.sqlite";
  int rc = sqlite3_open(db_path.c_str(), &db);
  if (rc != SQLITE_OK) {
    throw EPublish("cannot open SQLite database: " + std::string(db_path),
                   EPublish::kFailSqlite);
  }

  // Begin transaction
  char* errmsg = NULL;
  rc = sqlite3_exec(db, "BEGIN TRANSACTION;", NULL, NULL, &errmsg);
  if (rc != SQLITE_OK) {
    sqlite3_close(db);
    throw EPublish("cannot begin SQLite transaction: " + std::string(errmsg),
                   EPublish::kFailSqlite);
  }

  // Determine the default gateway address
  std::string default_gateway_query = "SELECT address FROM gateway WHERE default_gw = 1 LIMIT 1;";
  sqlite3_stmt* stmt = NULL;

  rc = sqlite3_prepare_v2(db, default_gateway_query.c_str(), -1, &stmt, NULL);
  if (rc != SQLITE_OK) {
    sqlite3_exec(db, "ROLLBACK;", NULL, NULL, NULL);
    sqlite3_close(db);
    throw EPublish("cannot prepare SQLite statement: " + std::string(sqlite3_errmsg(db)),
                   EPublish::kFailSqlite);
  }

  std::string default_gw;
  if ((rc = sqlite3_step(stmt)) == SQLITE_ROW) {
    default_gw = std::string(reinterpret_cast<const char*>(sqlite3_column_text(stmt, 0)));
  } else if (rc != SQLITE_DONE) {
    sqlite3_finalize(stmt);
    sqlite3_exec(db, "ROLLBACK;", NULL, NULL, NULL);
    sqlite3_close(db);
    throw EPublish("error reading default gateway from SQLite database: " + std::string(sqlite3_errmsg(db)),
                   EPublish::kFailSqlite);
  }

  sqlite3_finalize(stmt);

  // Remove all entries from the gateway table
  std::string delete_query = "DELETE FROM gateway;";
  rc = sqlite3_exec(db, delete_query.c_str(), NULL, NULL, &errmsg);
  if (rc != SQLITE_OK) {
    sqlite3_exec(db, "ROLLBACK;", NULL, NULL, NULL);
    sqlite3_close(db);
    throw EPublish("cannot clear gateway table: " + std::string(errmsg),
                   EPublish::kFailSqlite);
  }

  bool default_gw_down{false};
  // Insert new entries into the gateway table, based off the json response
  while (gateway != NULL) {
    const JSON* address = JsonDocument::SearchInObject(gateway, "address", JSON_STRING);
    const JSON* status = JsonDocument::SearchInObject(gateway, "status", JSON_INT);

    if (address == NULL || status == NULL) {
      LogCvmfs(kLogUploadGateway, settings_.llvl | kLogStderr,
               "Invalid gateway entry: missing address or status");
      gateway = gateway->next_sibling;
      continue;
    }

    if (address->string_value == default_gw && status->int_value >= kGatewayMaintenance) {
      // If the default gateway is down, mark this so the new default gateway can be set
      default_gw_down = true;
    }

    std::string insert_query = "INSERT INTO gateway (address, status) VALUES ('" +
                               std::string(address->string_value) + "', '" +
                               StringifyInt(status->int_value) + "');";

    rc = sqlite3_exec(db, insert_query.c_str(), NULL, NULL, &errmsg);
    if (rc != SQLITE_OK) {
      sqlite3_exec(db, "ROLLBACK;", NULL, NULL, NULL);
      sqlite3_close(db);
      throw EPublish("cannot insert into gateway table: " + std::string(errmsg),
                     EPublish::kFailSqlite);
    }

    gateway = gateway->next_sibling;
  }

  // Search for the default gateway address in the database, and update the default_gw column
  if (!default_gw.empty()) {
    std::string update_query = "UPDATE gateway SET default_gw = 1 WHERE address = '" +
                               default_gw + "';";
    rc = sqlite3_exec(db, update_query.c_str(), NULL, NULL, &errmsg);
    if (rc != SQLITE_OK) {
      sqlite3_exec(db, "ROLLBACK;", NULL, NULL, NULL);
      sqlite3_close(db);
      throw EPublish("cannot update default gateway: " + std::string(errmsg),
                     EPublish::kFailSqlite);
    }
    int rowsAffected = sqlite3_changes(db);
    if (rowsAffected == 0 || default_gw_down) {
      // No row affected, so default gateway has been removed, or its status is set to down.
      // Reset the default gateway to 0
      std::string reset_default_query = "UPDATE gateway SET default_gw = 0;";
      rc = sqlite3_exec(db, reset_default_query.c_str(), NULL, NULL, &errmsg);
      if (rc != SQLITE_OK) {
      sqlite3_exec(db, "ROLLBACK;", NULL, NULL, NULL);
      sqlite3_close(db);
      throw EPublish("cannot reset default gateway: " + std::string(errmsg),
               EPublish::kFailSqlite);
      }
      // Set a new default gateway: the first gateway with status 0
      std::string set_default_query = "UPDATE gateway SET default_gw = 1 WHERE rowid = (SELECT rowid FROM gateway WHERE status = " + std::to_string(kGatewayUp) + " LIMIT 1);";
      rc = sqlite3_exec(db, set_default_query.c_str(), NULL, NULL, &errmsg);
      if (rc != SQLITE_OK) {
      sqlite3_exec(db, "ROLLBACK;", NULL, NULL, NULL);
      sqlite3_close(db);
      throw EPublish("cannot set new default gateway: " + std::string(errmsg),
               EPublish::kFailSqlite);
      }
      LogCvmfs(kLogUploadGateway, settings_.llvl | kLogStderr,
           "No default gateway found in the database. Set new default gateway.");
    }
  }

  // Commit transaction
  rc = sqlite3_exec(db, "COMMIT;", NULL, NULL, &errmsg);
  if (rc != SQLITE_OK) {
    sqlite3_exec(db, "ROLLBACK;", NULL, NULL, NULL);
    sqlite3_close(db);
    throw EPublish("cannot commit SQLite transaction: " + std::string(errmsg),
                   EPublish::kFailSqlite);
  }

  sqlite3_close(db);

  return false;
}

/**
 * Read the gateway addresses from the local token ring SQLite database
 * @param repo_path The path to the repository
 * @param addresses The vector to store the addresses
 */
void Publisher::Session::GetRingGwsByPriority(const std::string &repo_path, std::vector<std::string>& addresses) const {
  sqlite3* db = NULL;

  // Extract the repo name from the repo path
  size_t end_pos = repo_path.find('/', 0);
  if (end_pos == 0) {
    throw EPublish("repo_path cannot start with a '/'", EPublish::kFailInput);
  }
  if (end_pos == std::string::npos) {
    throw EPublish("repo_path must contain a '/'", EPublish::kFailInput);
  }

  std::string repo_name = repo_path.substr(0, end_pos);
  string db_path = "/var/spool/cvmfs/" + repo_name + "/tokenring.sqlite";

  int rc = sqlite3_open(db_path.c_str(), &db);
  if (rc != SQLITE_OK) {
    throw EPublish("cannot open SQLite database: " + std::string(db_path),
                   EPublish::kFailSqlite);
  }

  std::string query = "SELECT address FROM gateway WHERE status <= " + std::to_string(kGatewayHighLoad) + " ORDER BY default_gw DESC, status ASC;";
  LogCvmfs(kLogPublish, kLogStderr, "Session acquire: query: %s", query.c_str());
  sqlite3_stmt* stmt = NULL;

  rc = sqlite3_prepare_v2(db, query.c_str(), -1, &stmt, NULL);
  if (rc != SQLITE_OK) {
    sqlite3_close(db);
    throw EPublish("cannot prepare SQLite statement: " + std::string(sqlite3_errmsg(db)),
                   EPublish::kFailSqlite);
  }

  while ((rc = sqlite3_step(stmt)) == SQLITE_ROW) {
    const unsigned char* addr = sqlite3_column_text(stmt, 0);
    if (addr) {
       addresses.push_back(reinterpret_cast<const char*>(addr));
    }
  }

  if (rc != SQLITE_DONE) {
    sqlite3_finalize(stmt);
    sqlite3_close(db);
    throw EPublish("error reading from SQLite database: " + std::string(sqlite3_errmsg(db)),
                   EPublish::kFailSqlite);
  }

  sqlite3_finalize(stmt);
  sqlite3_close(db);
}

/**
 * Set the current gateway address in the local token ring SQLite database
 * @param repo_path The path to the repository
 * @param address The address of the gateway to set as current
 */
void Publisher::Session::SetCurrentGw(const std::string &repo_path, const std::string &gw_address) const {
  // Extract the repo name from the repo path
  size_t end_pos = repo_path.find('/', 0);
  if (end_pos == 0) {
    throw EPublish("repo_path cannot start with a '/'", EPublish::kFailInput);
  }
  if (end_pos == std::string::npos) {
    throw EPublish("repo_path must contain a '/'", EPublish::kFailInput);
  }

  std::string repo_name = repo_path.substr(0, end_pos);
  std::string db_path = "/var/spool/cvmfs/" + repo_name + "/tokenring.sqlite";

  sqlite3* db = NULL;
  int rc = sqlite3_open(db_path.c_str(), &db);
  if (rc != SQLITE_OK) {
    throw EPublish("cannot open SQLite database: " + std::string(db_path),
                   EPublish::kFailSqlite);
  }

  std::string query = "UPDATE gateway SET current_gw = 1 WHERE address = '" + gw_address + "';";
  char* errmsg = NULL;
  rc = sqlite3_exec(db, query.c_str(), NULL, NULL, &errmsg);
  if (rc != SQLITE_OK) {
    sqlite3_close(db);
    throw EPublish("cannot update current gateway: " + std::string(errmsg),
                   EPublish::kFailSqlite);
  }

  sqlite3_close(db);
}

/**
 * Reset the current_gw bool for all gateways in the local token ring SQLite database
 * @param repo_path The path to the repository
 */
void Publisher::Session::ResetCurrentGw(const std::string &repo_path) const {
  // Extract the repo name from the repo path
  size_t end_pos = repo_path.find('/', 0);
  if (end_pos == 0) {
    throw EPublish("repo_path cannot start with a '/'", EPublish::kFailInput);
  }
  if (end_pos == std::string::npos) {
    throw EPublish("repo_path must contain a '/'", EPublish::kFailInput);
  }

  std::string repo_name = repo_path.substr(0, end_pos);
  std::string db_path = "/var/spool/cvmfs/" + repo_name + "/tokenring.sqlite";

  sqlite3* db = NULL;
  int rc = sqlite3_open(db_path.c_str(), &db);
  if (rc != SQLITE_OK) {
    throw EPublish("cannot open SQLite database: " + std::string(db_path),
                   EPublish::kFailSqlite);
  }

  std::string query = "UPDATE gateway SET current_gw = 0;";
  char* errmsg = NULL;
  rc = sqlite3_exec(db, query.c_str(), NULL, NULL, &errmsg);
  if (rc != SQLITE_OK) {
    sqlite3_close(db);
    throw EPublish("cannot reset current gateway: " + std::string(errmsg),
                   EPublish::kFailSqlite);
  }

  sqlite3_close(db);
}

std::string Publisher::Session::GetCurrentGw(const std::string &repo_path) const {
  // Extract the repo name from the repo path
  size_t end_pos = repo_path.find('/', 0);
  if (end_pos == 0) {
    throw EPublish("repo_path cannot start with a '/'", EPublish::kFailInput);
  }
  if (end_pos == std::string::npos) {
    throw EPublish("repo_path must contain a '/'", EPublish::kFailInput);
  }

  std::string repo_name = repo_path.substr(0, end_pos);
  std::string db_path = "/var/spool/cvmfs/" + repo_name + "/tokenring.sqlite";

  sqlite3* db = NULL;
  int rc = sqlite3_open(db_path.c_str(), &db);
  if (rc != SQLITE_OK) {
    throw EPublish("cannot open SQLite database: " + std::string(db_path),
                   EPublish::kFailSqlite);
  }

  std::string query = "SELECT address FROM gateway WHERE current_gw = 1;";
  sqlite3_stmt* stmt = NULL;

  rc = sqlite3_prepare_v2(db, query.c_str(), -1, &stmt, NULL);
  if (rc != SQLITE_OK) {
    sqlite3_close(db);
    throw EPublish("cannot prepare SQLite statement: " + std::string(sqlite3_errmsg(db)),
                   EPublish::kFailSqlite);
  }

  std::string gw_address;
  while ((rc = sqlite3_step(stmt)) == SQLITE_ROW) {
    const unsigned char* addr = sqlite3_column_text(stmt, 0);
    if (addr) {
       gw_address = reinterpret_cast<const char*>(addr);
    }
  }

  if (rc != SQLITE_DONE) {
    sqlite3_finalize(stmt);
    sqlite3_close(db);
    throw EPublish("error reading from SQLite database: " + std::string(sqlite3_errmsg(db)),
                   EPublish::kFailSqlite);
  }

  sqlite3_finalize(stmt);
  sqlite3_close(db);

  return gw_address;
}

LeaseReply Publisher::Session::AcquireInSingleGw(const gateway::GatewayKey &gw_key, std::string &session_token) {
  CurlBuffer buffer;
  MakeAcquireRequest(gw_key, settings_.repo_path, settings_.service_endpoint,
                       settings_.llvl, &buffer);

  return ParseAcquireReply(buffer, &session_token, settings_.llvl);
}

LeaseReply Publisher::Session::AcquireInMultiGw(const gateway::GatewayKey &gw_key, std::string &session_token) {
  std::string initial_endpoint = settings_.service_endpoint;
  // Update gateway db
  LogCvmfs(kLogPublish, kLogStderr, "Updating gateway database");
  std::vector<std::string> addresses;
  GetRingGwsByPriority(settings_.repo_path, addresses);
  if (addresses.empty()) {
    throw EPublish("cannot read gateway addresses", EPublish::kFailGatewayKey);
  }
  bool gwUnavailable{true};
  // As long as 1. not all gateway addresses have been attempted and 2. The reason the update request failed is because of an unavailable gateway
  for (int i = 0; i < addresses.size() && gwUnavailable; ++i) {
    gwUnavailable = UpdateGatewayDb(settings_.repo_path, addresses[i]);
  }
  
  // Retrieve gateway address and make lease acquire request
  CurlBuffer buffer;
  gwUnavailable = true;

  addresses.clear();
  GetRingGwsByPriority(settings_.repo_path, addresses);

  // As long as 1. not all gateway addresses have been attempted and 2. The reason the lease request failed is because of an unavailable gateway
  int i;
  for (i = 0; i < addresses.size() && gwUnavailable; ++i) {
    // Loop, starting at the start index (where our 'main' gateway is), and wrap around using modulo.
    LogCvmfs(kLogPublish, kLogStderr, "attempted publish address: %s", addresses[i].c_str());
    settings_.service_endpoint = addresses[i];
    gwUnavailable = MakeAcquireRequest(gw_key, settings_.repo_path, settings_.service_endpoint,
                       settings_.llvl, &buffer);
  }
  if (gwUnavailable) {
    // If gateway is still unavailable, this means all gateway addresses have been attempted, but none are available
    throw EPublish("cannot acquire lease from any gateway", EPublish::kFailLeaseHttp);
  }
  SetCurrentGw(settings_.repo_path, addresses[i-1]); // Decrement becaues the loop increments i after the last successful request
  return ParseAcquireReply(buffer, &session_token, settings_.llvl);
}

void Publisher::Session::Acquire(bool multi_gateway) {
  if (has_lease_)
    return;

  gateway::GatewayKey gw_key = gateway::ReadGatewayKey(settings_.gw_key_path);
  if (!gw_key.IsValid()) {
    throw EPublish("cannot read gateway key: " + settings_.gw_key_path,
                   EPublish::kFailGatewayKey);
  }

  ResetCurrentGw(settings_.repo_path);

  std::string session_token;
  LeaseReply rep;
  if (multi_gateway) {
    rep = AcquireInMultiGw(gw_key, session_token);
  } else {
    rep = AcquireInSingleGw(gw_key, session_token);
  }

  switch (rep) {
    case kLeaseReplySuccess:
      {
        has_lease_ = true;
        bool rvb = SafeWriteToFile(
          session_token,
          settings_.token_path,
          0600);
        if (!rvb) {
          throw EPublish("cannot write session token: " + settings_.token_path);
        }
      }
      break;
    case kLeaseReplyBusy:
      throw EPublish("lease path busy", EPublish::kFailLeaseBusy);
      break;
    case kLeaseReplyFailure:
    default:
      throw EPublish("cannot parse session token", EPublish::kFailLeaseBody);
  }
}

void Publisher::Session::Drop() {
  if (!has_lease_)
    return;
  // TODO(jblomer): there might be a better way to distinguish between the
  // nop-session and a real session
  if (settings_.service_endpoint.empty())
    return;

  std::string token;
  int fd_token = open(settings_.token_path.c_str(), O_RDONLY);
  bool rvb = SafeReadToString(fd_token, &token);
  close(fd_token);
  if (!rvb) {
    throw EPublish("cannot read session token: " + settings_.token_path,
                   EPublish::kFailGatewayKey);
  }
  gateway::GatewayKey gw_key = gateway::ReadGatewayKey(settings_.gw_key_path);
  if (!gw_key.IsValid()) {
    throw EPublish("cannot read gateway key: " + settings_.gw_key_path,
                   EPublish::kFailGatewayKey);
  }

  CurlBuffer buffer;
  std::string address = GetCurrentGw(settings_.repo_path);
  if (address.empty()) {
    throw EPublish("cannot read current gateway address", EPublish::kFailGatewayKey);
  }
  LogCvmfs(kLogPublish, kLogStderr, "attempted abort address: %s", address.c_str());
  // Make the drop request at the default gateway
  bool gwUnavailable = MakeDropRequest(gw_key, token, address, settings_.llvl, &buffer);
  LeaseReply rep;
  if (gwUnavailable) {
    // If gateway is unavailable, treat it as a success and drop the lease
    rep = kLeaseReplySuccess;    
  } else {
    rep = ParseDropReply(buffer, settings_.llvl);
  }
  int rvi = 0;
  switch (rep) {
    case kLeaseReplySuccess:
      has_lease_ = false;
      rvi = unlink(settings_.token_path.c_str());
      if (rvi != 0)
        throw EPublish("cannot delete session token " + settings_.token_path);
      break;
    case kLeaseReplyFailure:
    default:
      throw EPublish("gateway doesn't recognize the lease or cannot drop it",
                     EPublish::kFailLeaseBody);
  }
}

Publisher::Session::~Session() {
  if (keep_alive_)
    return;

  Drop();
}

}  // namespace publish
