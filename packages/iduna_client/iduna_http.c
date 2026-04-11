#include "iduna_http.h"

#include <curl/curl.h>
#include <stdio.h>
#include <string.h>

typedef struct {
    char *buf;
    size_t cap;
    size_t len;
} IdunaHttpBuffer;

static size_t iduna_http_write_cb(void *contents, size_t size, size_t nmemb, void *userp) {
    size_t total = size * nmemb;
    IdunaHttpBuffer *dst = (IdunaHttpBuffer *)userp;
    if (!dst || !dst->buf || dst->cap == 0) return 0;

    size_t available = (dst->cap - 1 > dst->len) ? (dst->cap - 1 - dst->len) : 0;
    size_t copy = total < available ? total : available;
    if (copy > 0) {
        memcpy(dst->buf + dst->len, contents, copy);
        dst->len += copy;
        dst->buf[dst->len] = '\0';
    }
    return total;
}

static int iduna_http_request(const char *method,
                              const char *url,
                              const char *json_body,
                              const char *auth_bearer_token,
                              char *out_body,
                              size_t out_body_len,
                              long *out_http_status) {
    static int curl_ready = 0;
    if (!curl_ready) {
        if (curl_global_init(CURL_GLOBAL_DEFAULT) != CURLE_OK) return -1;
        curl_ready = 1;
    }

    if (!url || !out_body || out_body_len == 0) return -1;
    out_body[0] = '\0';
    if (out_http_status) *out_http_status = 0;

    CURL *curl = curl_easy_init();
    if (!curl) return -1;

    IdunaHttpBuffer dst = { out_body, out_body_len, 0 };
    struct curl_slist *headers = NULL;
    headers = curl_slist_append(headers, "Content-Type: application/json");
    headers = curl_slist_append(headers, "Accept: application/json");

    char auth_header[2300];
    auth_header[0] = '\0';
    if (auth_bearer_token && auth_bearer_token[0]) {
        snprintf(auth_header, sizeof(auth_header), "Authorization: Bearer %s", auth_bearer_token);
        headers = curl_slist_append(headers, auth_header);
    }

    curl_easy_setopt(curl, CURLOPT_URL, url);
    curl_easy_setopt(curl, CURLOPT_HTTPHEADER, headers);
    curl_easy_setopt(curl, CURLOPT_USERAGENT, "Kikoryu/VS1");
    curl_easy_setopt(curl, CURLOPT_CONNECTTIMEOUT, 5L);
    curl_easy_setopt(curl, CURLOPT_TIMEOUT, 10L);
    curl_easy_setopt(curl, CURLOPT_WRITEFUNCTION, iduna_http_write_cb);
    curl_easy_setopt(curl, CURLOPT_WRITEDATA, &dst);

    if (strcmp(method, "POST") == 0) {
        curl_easy_setopt(curl, CURLOPT_POST, 1L);
        curl_easy_setopt(curl, CURLOPT_POSTFIELDS, json_body ? json_body : "{}");
    }

    CURLcode res = curl_easy_perform(curl);
    if (res == CURLE_OK) {
        long status = 0;
        curl_easy_getinfo(curl, CURLINFO_RESPONSE_CODE, &status);
        if (out_http_status) *out_http_status = status;
    }

    curl_slist_free_all(headers);
    curl_easy_cleanup(curl);
    return (res == CURLE_OK) ? 0 : -1;
}

int iduna_http_post_json(const char *url,
                         const char *json_body,
                         const char *auth_bearer_token,
                         char *out_body,
                         size_t out_body_len,
                         long *out_http_status) {
    return iduna_http_request("POST", url, json_body, auth_bearer_token, out_body, out_body_len, out_http_status);
}

int iduna_http_get(const char *url,
                   const char *auth_bearer_token,
                   char *out_body,
                   size_t out_body_len,
                   long *out_http_status) {
    return iduna_http_request("GET", url, NULL, auth_bearer_token, out_body, out_body_len, out_http_status);
}
