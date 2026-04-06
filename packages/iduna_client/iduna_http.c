#include "iduna_http.h"

#include <curl/curl.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <ctype.h>

typedef struct {
    char *buf;
    size_t cap;
    size_t len;
} IdunaHttpBuffer;

static size_t iduna_http_write_cb(void *contents, size_t size, size_t nmemb, void *userp) {
    IdunaHttpBuffer *dst = (IdunaHttpBuffer *)userp;
    size_t total = size * nmemb;
    if (!dst || !dst->buf || dst->cap == 0) return total;
    size_t copy = total;
    if (dst->len + copy >= dst->cap) {
        copy = dst->cap - dst->len - 1;
    }
    if (copy > 0) {
        memcpy(dst->buf + dst->len, contents, copy);
        dst->len += copy;
        dst->buf[dst->len] = '\0';
    }
    return total;
}

static int iduna_http_common(const char *url,
                             const char *method,
                             const char *json_body,
                             const char *bearer_token,
                             char *out_buf,
                             size_t out_buf_sz,
                             long *out_status) {
    static int curl_bootstrapped = 0;
    if (!curl_bootstrapped) {
        curl_global_init(CURL_GLOBAL_DEFAULT);
        curl_bootstrapped = 1;
    }

    if (!url || !method || !out_buf || out_buf_sz == 0) return 0;
    out_buf[0] = '\0';

    CURL *curl = curl_easy_init();
    if (!curl) return 0;

    struct curl_slist *headers = NULL;
    headers = curl_slist_append(headers, "User-Agent: Kikoryu/VS1");
    if (strcmp(method, "POST") == 0) {
        headers = curl_slist_append(headers, "Content-Type: application/json");
    }
    if (bearer_token && bearer_token[0]) {
        char auth_header[2300];
        snprintf(auth_header, sizeof(auth_header), "Authorization: Bearer %s", bearer_token);
        headers = curl_slist_append(headers, auth_header);
    }

    IdunaHttpBuffer dst = {out_buf, out_buf_sz, 0};
    curl_easy_setopt(curl, CURLOPT_URL, url);
    curl_easy_setopt(curl, CURLOPT_CUSTOMREQUEST, method);
    curl_easy_setopt(curl, CURLOPT_HTTPHEADER, headers);
    curl_easy_setopt(curl, CURLOPT_CONNECTTIMEOUT, 5L);
    curl_easy_setopt(curl, CURLOPT_TIMEOUT, 10L);
    curl_easy_setopt(curl, CURLOPT_FOLLOWLOCATION, 1L);
    curl_easy_setopt(curl, CURLOPT_WRITEFUNCTION, iduna_http_write_cb);
    curl_easy_setopt(curl, CURLOPT_WRITEDATA, &dst);
    if (strcmp(method, "POST") == 0) {
        curl_easy_setopt(curl, CURLOPT_POSTFIELDS, json_body ? json_body : "{}");
    }

    CURLcode res = curl_easy_perform(curl);
    long code = 0;
    curl_easy_getinfo(curl, CURLINFO_RESPONSE_CODE, &code);
    if (out_status) *out_status = code;

    curl_slist_free_all(headers);
    curl_easy_cleanup(curl);

    return res == CURLE_OK;
}

int iduna_http_post_json(const char *url,
                         const char *json_body,
                         const char *bearer_token,
                         char *out_buf,
                         size_t out_buf_sz,
                         long *out_status) {
    return iduna_http_common(url, "POST", json_body, bearer_token, out_buf, out_buf_sz, out_status);
}

int iduna_http_get(const char *url,
                   const char *bearer_token,
                   char *out_buf,
                   size_t out_buf_sz,
                   long *out_status) {
    return iduna_http_common(url, "GET", NULL, bearer_token, out_buf, out_buf_sz, out_status);
}

static const char *iduna_find_key(const char *json, const char *key) {
    if (!json || !key) return NULL;
    char pattern[128];
    snprintf(pattern, sizeof(pattern), "\"%s\"", key);
    const char *p = strstr(json, pattern);
    if (!p) return NULL;
    p += strlen(pattern);
    while (*p && (*p == ' ' || *p == '\t' || *p == '\r' || *p == '\n')) p++;
    if (*p != ':') return NULL;
    p++;
    while (*p && (*p == ' ' || *p == '\t' || *p == '\r' || *p == '\n')) p++;
    return p;
}

int iduna_json_get_string(const char *json,
                          const char *key,
                          char *out,
                          size_t out_sz) {
    if (!out || out_sz == 0) return 0;
    out[0] = '\0';
    const char *p = iduna_find_key(json, key);
    if (!p || *p != '"') return 0;
    p++;
    size_t i = 0;
    while (*p && *p != '"') {
        if (*p == '\\' && p[1]) {
            p++;
        }
        if (i + 1 < out_sz) {
            out[i++] = *p;
        }
        p++;
    }
    out[i] = '\0';
    return *p == '"';
}

int iduna_json_get_int(const char *json, const char *key, int *out_value) {
    const char *p = iduna_find_key(json, key);
    if (!p || !out_value) return 0;
    char num[32];
    size_t i = 0;
    if (*p == '-') {
        num[i++] = *p++;
    }
    while (*p && isdigit((unsigned char)*p) && i + 1 < sizeof(num)) {
        num[i++] = *p++;
    }
    num[i] = '\0';
    if (i == 0 || (i == 1 && num[0] == '-')) return 0;
    *out_value = atoi(num);
    return 1;
}
