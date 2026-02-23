<template>
  <div id="reader">
    <!-- Header -->
    <div class="header">
      <div class="header-left">
        <button class="btn-toggle" v-on:click="toggleSidebar" v-if="book">
          <span class="icon-menu">&#9776;</span>
        </button>
        <span class="header-title">eBook Reader</span>
      </div>
      <div class="header-right" v-if="book">
        <span class="header-info">{{ book.title }}</span>
      </div>
    </div>

    <div class="main">
      <!-- Sidebar -->
      <div class="sidebar" v-if="book" v-show="sidebarOpen">
        <div class="sidebar-header">
          <div class="book-cover" v-if="book.coverUrl">
            <img :src="book.coverUrl" alt="cover">
          </div>
          <h2 class="book-title">{{ book.title }}</h2>
          <p class="book-author">{{ book.author }}</p>
          <p class="book-format">{{ book.format.toUpperCase() }} · {{ book.chapters.length }} chapters</p>
        </div>
        <div class="sidebar-divider"></div>
        <ul class="toc">
          <li
            v-for="ch in book.chapters"
            :key="ch.id"
            :class="{ active: ch.id === currentChapter }"
            v-on:click="loadChapter(ch.id)"
          >
            <span class="toc-num">{{ ch.id + 1 }}</span>
            <span class="toc-title">{{ ch.title }}</span>
          </li>
        </ul>
      </div>

      <!-- Content -->
      <div class="content">
        <!-- Loading -->
        <div v-if="loading" class="status-page">
          <div class="spinner"></div>
          <p class="status-text">Loading book...</p>
        </div>

        <!-- Error -->
        <div v-else-if="error" class="status-page">
          <div class="status-icon error-icon">&#10007;</div>
          <p class="status-text error-text">{{ error }}</p>
          <button class="btn-retry" v-on:click="loadBook">Retry</button>
        </div>

        <!-- Empty -->
        <div v-else-if="!book" class="status-page welcome">
          <div class="welcome-icon">&#128214;</div>
          <h2 class="welcome-title">eBook Reader</h2>
          <p class="welcome-desc">Pass <code>?file=URL</code> to open a book.</p>
          <div class="welcome-example">
            <code>?file=https://example.com/book.epub</code>
          </div>
          <p class="welcome-formats">Supported: EPUB, TXT</p>
        </div>

        <!-- Chapter loading -->
        <div v-else-if="chapterLoading" class="status-page">
          <div class="spinner"></div>
          <p class="status-text">Loading chapter...</p>
        </div>

        <!-- Reader -->
        <iframe
          v-else
          ref="reader"
          class="reader-frame"
          frameborder="0"
          sandbox="allow-same-origin"
        ></iframe>
      </div>
    </div>

    <!-- Chapter nav -->
    <div class="footer" v-if="book && book.chapters.length > 0">
      <button
        class="btn-nav"
        v-on:click="prevChapter"
        :disabled="currentChapter <= 0"
      >&laquo; Prev</button>
      <span class="footer-info">{{ currentChapter + 1 }} / {{ book.chapters.length }}</span>
      <button
        class="btn-nav"
        v-on:click="nextChapter"
        :disabled="currentChapter >= book.chapters.length - 1"
      >Next &raquo;</button>
    </div>
  </div>
</template>

<script>
function request(url, cb) {
  var xhr = new XMLHttpRequest()
  xhr.open('GET', url, true)
  xhr.onreadystatechange = function () {
    if (xhr.readyState === 4) {
      if (xhr.status === 200) {
        try {
          cb(null, JSON.parse(xhr.responseText))
        } catch (e) {
          cb(e, null)
        }
      } else {
        cb(new Error('HTTP ' + xhr.status), null)
      }
    }
  }
  xhr.send()
}

export default {
  name: 'App',
  data: function () {
    return {
      book: null,
      currentChapter: -1,
      loading: false,
      chapterLoading: false,
      error: null,
      fileURL: '',
      sidebarOpen: true
    }
  },
  mounted: function () {
    var params = this.getQueryParams()
    if (params.file) {
      this.fileURL = params.file
      this.loadBook()
    }
  },
  methods: {
    getQueryParams: function () {
      var search = window.location.search.substring(1)
      var params = {}
      if (!search) return params
      var pairs = search.split('&')
      for (var i = 0; i < pairs.length; i++) {
        var kv = pairs[i].split('=')
        params[decodeURIComponent(kv[0])] = decodeURIComponent(kv[1] || '')
      }
      return params
    },
    toggleSidebar: function () {
      this.sidebarOpen = !this.sidebarOpen
    },
    loadBook: function () {
      var self = this
      self.loading = true
      self.error = null
      request('/api/book/meta?file=' + encodeURIComponent(self.fileURL), function (err, data) {
        self.loading = false
        if (err) {
          self.error = 'Failed to load book: ' + err.message
          return
        }
        self.book = data
        if (data.chapters && data.chapters.length > 0) {
          self.loadChapter(0)
        }
      })
    },
    loadChapter: function (id) {
      var self = this
      self.currentChapter = id
      self.chapterLoading = true
      request('/api/book/chapter/' + id + '?file=' + encodeURIComponent(self.fileURL), function (err, data) {
        self.chapterLoading = false
        if (err) {
          self.error = 'Failed to load chapter: ' + err.message
          return
        }
        self.$nextTick(function () {
          var iframe = self.$refs.reader
          if (iframe) {
            var doc = iframe.contentDocument || iframe.contentWindow.document
            var html = '<!DOCTYPE html><html><head>'
            html += '<meta charset="utf-8">'
            html += '<style>'
            html += 'body{font-family:-apple-system,Segoe UI,Helvetica,Arial,sans-serif;'
            html += 'line-height:1.8;color:#333;max-width:720px;margin:0 auto;padding:24px 32px;'
            html += 'font-size:16px;}'
            html += 'p{margin:0 0 1em 0;text-indent:2em;}'
            html += 'h1,h2,h3{margin:1.2em 0 0.6em;color:#222;}'
            html += 'img{max-width:100%;height:auto;}'
            html += 'a{color:#4a7ebb;text-decoration:none;}'
            html += '</style></head><body>'
            html += data.content
            html += '</body></html>'
            doc.open()
            doc.write(html)
            doc.close()
          }
        })
      })
    },
    prevChapter: function () {
      if (this.currentChapter > 0) {
        this.loadChapter(this.currentChapter - 1)
      }
    },
    nextChapter: function () {
      if (this.book && this.currentChapter < this.book.chapters.length - 1) {
        this.loadChapter(this.currentChapter + 1)
      }
    }
  }
}
</script>

<style>
* {
  margin: 0;
  padding: 0;
  box-sizing: border-box;
}
html, body, #app, #reader {
  height: 100%;
  width: 100%;
  overflow: hidden;
}
#reader {
  display: table;
  width: 100%;
  height: 100%;
  font-family: -apple-system, "Segoe UI", Helvetica, Arial, sans-serif;
  color: #333;
  background: #fafafa;
}

/* Header */
.header {
  display: table-row;
  height: 48px;
}
.header > * {
  display: table-cell;
  vertical-align: middle;
  background: #fff;
  border-bottom: 1px solid #e5e5e5;
  padding: 0 16px;
}
.header-left {
  white-space: nowrap;
}
.header-right {
  text-align: right;
  color: #888;
  font-size: 13px;
}
.header-title {
  font-size: 15px;
  font-weight: 600;
  color: #333;
  margin-left: 8px;
}
.btn-toggle {
  background: none;
  border: 1px solid #ddd;
  border-radius: 4px;
  padding: 4px 8px;
  cursor: pointer;
  font-size: 16px;
  color: #555;
  vertical-align: middle;
}
.btn-toggle:hover {
  background: #f0f0f0;
}

/* Main area */
.main {
  display: table-row;
  height: 100%;
}
.main > * {
  display: table-cell;
  vertical-align: top;
}

/* Sidebar */
.sidebar {
  width: 280px;
  background: #fff;
  border-right: 1px solid #e5e5e5;
  overflow-y: auto;
  height: 100%;
}
.sidebar-header {
  padding: 20px 16px 12px;
  text-align: center;
}
.book-cover {
  margin-bottom: 12px;
}
.book-cover img {
  max-width: 120px;
  max-height: 160px;
  border-radius: 4px;
  box-shadow: 0 2px 8px rgba(0,0,0,0.15);
}
.book-title {
  font-size: 15px;
  font-weight: 600;
  color: #222;
  margin-bottom: 4px;
}
.book-author {
  font-size: 13px;
  color: #888;
  margin-bottom: 4px;
}
.book-format {
  font-size: 12px;
  color: #aaa;
}
.sidebar-divider {
  height: 1px;
  background: #eee;
  margin: 8px 16px;
}
.toc {
  list-style: none;
  padding: 4px 8px 16px;
}
.toc li {
  padding: 8px 12px;
  cursor: pointer;
  font-size: 13px;
  color: #555;
  border-radius: 6px;
  margin-bottom: 2px;
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
}
.toc li:hover {
  background: #f5f5f5;
}
.toc li.active {
  background: #eef4ff;
  color: #2563eb;
  font-weight: 500;
}
.toc-num {
  display: inline-block;
  width: 28px;
  color: #bbb;
  font-size: 12px;
}
.toc li.active .toc-num {
  color: #93b4f5;
}

/* Content */
.content {
  height: 100%;
  background: #fafafa;
}

/* Status pages */
.status-page {
  padding: 80px 40px;
  text-align: center;
}
.spinner {
  display: inline-block;
  width: 32px;
  height: 32px;
  border: 3px solid #e5e5e5;
  border-top-color: #2563eb;
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
  margin-bottom: 16px;
}
@keyframes spin {
  to { transform: rotate(360deg); }
}
.status-text {
  font-size: 14px;
  color: #888;
}
.error-icon {
  font-size: 36px;
  color: #e53e3e;
  margin-bottom: 12px;
}
.error-text {
  color: #c53030;
}
.btn-retry {
  margin-top: 16px;
  padding: 8px 24px;
  background: #2563eb;
  color: #fff;
  border: none;
  border-radius: 6px;
  cursor: pointer;
  font-size: 14px;
}
.btn-retry:hover {
  background: #1d4ed8;
}

/* Welcome */
.welcome-icon {
  font-size: 48px;
  margin-bottom: 16px;
}
.welcome-title {
  font-size: 22px;
  font-weight: 600;
  color: #222;
  margin-bottom: 8px;
}
.welcome-desc {
  font-size: 14px;
  color: #888;
  margin-bottom: 16px;
}
.welcome-example {
  background: #f0f0f0;
  border-radius: 6px;
  padding: 12px 16px;
  display: inline-block;
  margin-bottom: 16px;
}
.welcome-example code {
  font-size: 12px;
  color: #555;
  word-break: break-all;
}
.welcome-formats {
  font-size: 12px;
  color: #aaa;
}

/* Reader iframe */
.reader-frame {
  width: 100%;
  height: 100%;
  border: none;
  background: #fff;
}

/* Footer nav */
.footer {
  display: table-row;
  height: 44px;
}
.footer > * {
  vertical-align: middle;
}
.footer {
  background: #fff;
  border-top: 1px solid #e5e5e5;
  text-align: center;
  padding: 0 16px;
  line-height: 44px;
}
.btn-nav {
  background: none;
  border: 1px solid #ddd;
  border-radius: 4px;
  padding: 4px 16px;
  cursor: pointer;
  font-size: 13px;
  color: #555;
}
.btn-nav:hover {
  background: #f5f5f5;
}
.btn-nav:disabled {
  color: #ccc;
  border-color: #eee;
  cursor: default;
}
.btn-nav:disabled:hover {
  background: none;
}
.footer-info {
  display: inline-block;
  margin: 0 24px;
  font-size: 13px;
  color: #888;
}
</style>
