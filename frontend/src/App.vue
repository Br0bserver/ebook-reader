<template>
  <div id="reader" :class="'theme-' + theme">
    <!-- Sidebar overlay -->
    <div class="sidebar-overlay" v-if="book && sidebarOpen" v-on:click="closeSidebar"></div>

    <!-- Header -->
    <div class="header">
      <div class="header-left">
        <button class="btn-icon" v-on:click="toggleSidebar" v-if="book" title="Menu">&#9776;</button>
        <span class="header-title">eBook Reader</span>
      </div>
      <div class="header-right" v-if="book">
        <button class="btn-icon" v-on:click="cycleFontSize" title="Font size">Aa</button>
        <button class="btn-icon" v-on:click="zoomOut" title="Zoom out">-</button>
        <span class="zoom-label">{{ zoomLevel }}%</span>
        <button class="btn-icon" v-on:click="zoomIn" title="Zoom in">+</button>
        <button class="btn-icon" v-on:click="cycleTheme" :title="'Theme: ' + theme">
          <span v-if="theme === 'light'">&#9788;</span>
          <span v-else-if="theme === 'dark'">&#9790;</span>
          <span v-else>&#9752;</span>
        </button>
      </div>
    </div>

    <div class="main">
      <!-- Sidebar -->
      <div class="sidebar" v-if="book" v-show="sidebarOpen" ref="sidebar">
        <div class="sidebar-header">
          <div class="book-cover" v-if="book.coverUrl && !coverError">
            <img :src="book.coverUrl" alt="cover" v-on:error="coverError = true">
          </div>
          <h2 class="book-title">{{ book.title }}</h2>
          <p class="book-author">{{ book.author }}</p>
          <p class="book-format">{{ book.format.toUpperCase() }} · {{ book.chapters.length }} chapters</p>
        </div>
        <div class="sidebar-divider"></div>
        <ul class="toc" ref="toc">
          <li
            v-for="ch in book.chapters"
            :key="ch.id"
            :ref="'toc-' + ch.id"
            :class="{ active: ch.id === currentChapter }"
            v-on:click="selectChapter(ch.id)"
          >
            <span class="toc-num">{{ ch.id + 1 }}</span>
            <span class="toc-title">{{ ch.title }}</span>
          </li>
        </ul>
      </div>

      <!-- Content -->
      <div class="content">
        <div v-if="loading" class="status-page">
          <div class="spinner"></div>
          <p class="status-text">Loading book...</p>
        </div>
        <div v-else-if="error" class="status-page">
          <div class="status-icon error-icon">&#10007;</div>
          <p class="status-text error-text">{{ error }}</p>
          <button class="btn-action" v-on:click="loadBook">Retry</button>
        </div>
        <div v-else-if="!book" class="status-page welcome">
          <div class="welcome-icon">&#128214;</div>
          <h2 class="welcome-title">eBook Reader</h2>
          <p class="welcome-desc">Pass <code>?file=URL</code> to open a book.</p>
          <div class="welcome-example">
            <code>?file=https://example.com/book.epub</code>
          </div>
          <p class="welcome-formats">Supported: EPUB, TXT</p>
        </div>
        <div v-else-if="chapterLoading" class="status-page">
          <div class="spinner"></div>
          <p class="status-text">Loading chapter...</p>
        </div>
        <iframe
          v-else
          ref="reader"
          class="reader-frame"
          frameborder="0"
          sandbox="allow-same-origin"
        ></iframe>
      </div>
    </div>

    <!-- Footer -->
    <div class="footer" v-if="book && book.chapters.length > 0">
      <button class="btn-nav" v-on:click="prevChapter" :disabled="currentChapter <= 0">&laquo;</button>
      <span class="footer-info">{{ currentChapter + 1 }} / {{ book.chapters.length }}<span v-if="readPercent >= 0"> · {{ readPercent }}%</span></span>
      <button class="btn-nav" v-on:click="nextChapter" :disabled="currentChapter >= book.chapters.length - 1">&raquo;</button>
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
        try { cb(null, JSON.parse(xhr.responseText)) }
        catch (e) { cb(e, null) }
      } else {
        cb(new Error('HTTP ' + xhr.status), null)
      }
    }
  }
  xhr.send()
}

var ZOOM_LEVELS = [80, 90, 100, 110, 125, 150]
var FONT_SIZES = [14, 16, 20]
var THEMES = ['light', 'dark', 'sepia']
var THEME_STYLES = {
  light: 'body{background:#fff;color:#333;}a{color:#4a7ebb;}',
  dark: 'body{background:#1a1a2e;color:#ccc;}a{color:#7eb8da;}',
  sepia: 'body{background:#f5f0e1;color:#5b4636;}a{color:#7b6043;}'
}

function loadSetting(key, fallback) {
  try { var v = localStorage.getItem(key); return v !== null ? v : fallback }
  catch (e) { return fallback }
}
function saveSetting(key, val) {
  try { localStorage.setItem(key, val) } catch (e) {}
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
      sidebarOpen: false,
      coverError: false,
      readPercent: -1,
      theme: loadSetting('ebook_theme', 'light'),
      fontSize: parseInt(loadSetting('ebook_fontsize', '16'), 10),
      zoomLevel: parseInt(loadSetting('ebook_zoom', '100'), 10)
    }
  },
  mounted: function () {
    this.applyZoom()
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

    // Sidebar
    toggleSidebar: function () { this.sidebarOpen = !this.sidebarOpen },
    closeSidebar: function () { this.sidebarOpen = false },
    selectChapter: function (id) {
      this.loadChapter(id)
      this.sidebarOpen = false
    },

    // Theme
    cycleTheme: function () {
      var idx = THEMES.indexOf(this.theme)
      this.theme = THEMES[(idx + 1) % THEMES.length]
      saveSetting('ebook_theme', this.theme)
      this.updateIframeStyle()
    },

    // Font size
    cycleFontSize: function () {
      var idx = FONT_SIZES.indexOf(this.fontSize)
      this.fontSize = FONT_SIZES[(idx + 1) % FONT_SIZES.length]
      saveSetting('ebook_fontsize', String(this.fontSize))
      this.updateIframeStyle()
    },

    // Zoom
    zoomIn: function () {
      var idx = ZOOM_LEVELS.indexOf(this.zoomLevel)
      if (idx < ZOOM_LEVELS.length - 1) {
        this.zoomLevel = ZOOM_LEVELS[idx + 1]
        saveSetting('ebook_zoom', String(this.zoomLevel))
        this.applyZoom()
      }
    },
    zoomOut: function () {
      var idx = ZOOM_LEVELS.indexOf(this.zoomLevel)
      if (idx > 0) {
        this.zoomLevel = ZOOM_LEVELS[idx - 1]
        saveSetting('ebook_zoom', String(this.zoomLevel))
        this.applyZoom()
      }
    },
    applyZoom: function () {
      document.documentElement.style.zoom = (this.zoomLevel / 100)
    },

    // Book loading
    loadBook: function () {
      var self = this
      self.loading = true
      self.error = null
      self.coverError = false
      request('/api/book/meta?file=' + encodeURIComponent(self.fileURL), function (err, data) {
        self.loading = false
        if (err) {
          self.error = 'Failed to load book: ' + err.message
          return
        }
        self.book = data
        if (data.chapters && data.chapters.length > 0) {
          var saved = loadSetting('ebook_progress_' + data.id, '0')
          var startCh = parseInt(saved, 10) || 0
          if (startCh >= data.chapters.length) startCh = 0
          self.loadChapter(startCh)
        }
      })
    },

    loadChapter: function (id) {
      var self = this
      self.currentChapter = id
      self.chapterLoading = true
      self.readPercent = -1
      if (self.book) {
        saveSetting('ebook_progress_' + self.book.id, String(id))
      }
      self.scrollTocToActive()
      request('/api/book/chapter/' + id + '?file=' + encodeURIComponent(self.fileURL), function (err, data) {
        self.chapterLoading = false
        if (err) {
          self.error = 'Failed to load chapter: ' + err.message
          return
        }
        self.$nextTick(function () { self.renderChapter(data.content) })
      })
    },

    renderChapter: function (content) {
      var self = this
      var iframe = self.$refs.reader
      if (!iframe) return
      var doc = iframe.contentDocument || iframe.contentWindow.document
      var html = '<!DOCTYPE html><html><head><meta charset="utf-8">'
      html += '<style>'
      html += 'body{font-family:-apple-system,Segoe UI,Helvetica,Arial,sans-serif;'
      html += 'line-height:1.8;margin:0;padding:0;'
      html += 'font-size:' + self.fontSize + 'px;}'
      html += THEME_STYLES[self.theme]
      html += '.reader-wrap{max-width:80%;margin:0 auto;padding:24px 20px;}'
      html += 'p{margin:0 0 1em 0;text-indent:2em;}'
      html += 'h1,h2,h3{margin:1.2em 0 0.6em;}'
      html += 'img{max-width:100%;height:auto;}'
      html += 'a{text-decoration:none;}'
      html += '</style></head><body><div class="reader-wrap">'
      html += content
      html += '</div></body></html>'
      doc.open()
      doc.write(html)
      doc.close()
      // Bind scroll for read percent
      try {
        var win = iframe.contentWindow
        win.onscroll = function () { self.updateReadPercent(win) }
        // Initial percent
        setTimeout(function () { self.updateReadPercent(win) }, 100)
      } catch (e) {}
    },

    updateIframeStyle: function () {
      var iframe = this.$refs.reader
      if (!iframe) return
      try {
        var doc = iframe.contentDocument || iframe.contentWindow.document
        var body = doc.body
        if (!body) return
        // Update font size
        body.style.fontSize = this.fontSize + 'px'
        // Update theme: remove old style, add new
        var old = doc.getElementById('theme-style')
        if (old) old.parentNode.removeChild(old)
        var style = doc.createElement('style')
        style.id = 'theme-style'
        style.type = 'text/css'
        if (style.styleSheet) {
          style.styleSheet.cssText = THEME_STYLES[this.theme]
        } else {
          style.appendChild(doc.createTextNode(THEME_STYLES[this.theme]))
        }
        doc.head.appendChild(style)
      } catch (e) {}
    },

    updateReadPercent: function (win) {
      try {
        var doc = win.document
        var body = doc.body
        var de = doc.documentElement
        var scrollTop = (de && de.scrollTop) || body.scrollTop || 0
        var scrollHeight = Math.max(
          body.scrollHeight || 0, de.scrollHeight || 0,
          body.offsetHeight || 0, de.offsetHeight || 0
        )
        var clientHeight = de.clientHeight || body.clientHeight || 0
        var total = scrollHeight - clientHeight
        if (total <= 0) {
          this.readPercent = 100
        } else {
          this.readPercent = Math.min(100, Math.round(scrollTop / total * 100))
        }
      } catch (e) {
        this.readPercent = -1
      }
    },

    // TOC auto scroll
    scrollTocToActive: function () {
      var self = this
      self.$nextTick(function () {
        var refName = 'toc-' + self.currentChapter
        var els = self.$refs[refName]
        var el = els && els[0] ? els[0] : els
        var container = self.$refs.toc
        if (el && container) {
          var top = el.offsetTop - container.offsetTop
          container.scrollTop = top - container.clientHeight / 2 + el.offsetHeight / 2
        }
      })
    },

    prevChapter: function () {
      if (this.currentChapter > 0) this.loadChapter(this.currentChapter - 1)
    },
    nextChapter: function () {
      if (this.book && this.currentChapter < this.book.chapters.length - 1) this.loadChapter(this.currentChapter + 1)
    }
  }
}
</script>

<style>
* { margin: 0; padding: 0; box-sizing: border-box; }
html, body, #app, #reader { height: 100%; width: 100%; overflow: hidden; }
#reader {
  position: relative;
  font-family: -apple-system, "Segoe UI", Helvetica, Arial, sans-serif;
  color: #333;
  background: #fafafa;
}

/* ===== Theme colors ===== */
.theme-light { background: #fafafa; color: #333; }
.theme-light .header, .theme-light .footer { background: #fff; border-color: #e5e5e5; }
.theme-light .sidebar { background: #fff; border-color: #e5e5e5; }
.theme-light .toc li:hover { background: #f5f5f5; }
.theme-light .toc li.active { background: #eef4ff; color: #2563eb; }

.theme-dark { background: #16162a; color: #ccc; }
.theme-dark .header, .theme-dark .footer { background: #1a1a2e; border-color: #2a2a4a; }
.theme-dark .sidebar { background: #1a1a2e; border-color: #2a2a4a; }
.theme-dark .sidebar-divider { background: #2a2a4a; }
.theme-dark .book-title, .theme-dark .header-title { color: #ddd; }
.theme-dark .book-author, .theme-dark .book-format { color: #888; }
.theme-dark .toc li { color: #aaa; }
.theme-dark .toc li:hover { background: #252545; }
.theme-dark .toc li.active { background: #1e2a4a; color: #7eb8da; }
.theme-dark .toc li.active .toc-num { color: #5a7a9a; }
.theme-dark .btn-icon { color: #aaa; border-color: #3a3a5a; }
.theme-dark .btn-icon:hover { background: #252545; }
.theme-dark .btn-nav { color: #aaa; border-color: #3a3a5a; }
.theme-dark .btn-nav:hover { background: #252545; }
.theme-dark .btn-nav:disabled { color: #555; border-color: #2a2a4a; }
.theme-dark .footer-info, .theme-dark .zoom-label { color: #888; }
.theme-dark .content { background: #16162a; }
.theme-dark .reader-frame { background: #1a1a2e; }
.theme-dark .sidebar-overlay { background: rgba(0,0,0,0.6); }

.theme-sepia { background: #ede4d3; color: #5b4636; }
.theme-sepia .header, .theme-sepia .footer { background: #f5f0e1; border-color: #d9cdb7; }
.theme-sepia .sidebar { background: #f5f0e1; border-color: #d9cdb7; }
.theme-sepia .sidebar-divider { background: #d9cdb7; }
.theme-sepia .book-title, .theme-sepia .header-title { color: #4a3728; }
.theme-sepia .toc li { color: #6b5744; }
.theme-sepia .toc li:hover { background: #ede4d3; }
.theme-sepia .toc li.active { background: #e6d9c3; color: #7b6043; }
.theme-sepia .btn-icon { color: #6b5744; border-color: #c9bda7; }
.theme-sepia .btn-icon:hover { background: #ede4d3; }
.theme-sepia .btn-nav { color: #6b5744; border-color: #c9bda7; }
.theme-sepia .footer-info, .theme-sepia .zoom-label { color: #8a7a66; }
.theme-sepia .content { background: #ede4d3; }
.theme-sepia .reader-frame { background: #f5f0e1; }

/* ===== Header ===== */
.header {
  position: absolute;
  top: 0; left: 0; right: 0;
  height: 48px;
  background: #fff;
  border-bottom: 1px solid #e5e5e5;
  z-index: 50;
  display: table;
  width: 100%;
}
.header > * {
  display: table-cell;
  vertical-align: middle;
  padding: 0 12px;
}
.header-left { white-space: nowrap; }
.header-right { text-align: right; white-space: nowrap; }
.header-title {
  font-size: 15px;
  font-weight: 600;
  color: #333;
  margin-left: 6px;
}
.btn-icon {
  background: none;
  border: 1px solid #ddd;
  border-radius: 4px;
  min-width: 36px;
  min-height: 36px;
  padding: 4px 8px;
  cursor: pointer;
  font-size: 15px;
  color: #555;
  vertical-align: middle;
  margin-left: 4px;
}
.btn-icon:hover { background: #f0f0f0; }
.zoom-label {
  font-size: 12px;
  color: #888;
  vertical-align: middle;
  margin: 0 2px;
}

/* ===== Main area ===== */
.main {
  position: absolute;
  top: 48px; bottom: 44px;
  left: 0; right: 0;
  overflow: hidden;
}

/* ===== Sidebar overlay ===== */
.sidebar-overlay {
  position: absolute;
  top: 0; left: 0; right: 0; bottom: 0;
  background: rgba(0,0,0,0.35);
  z-index: 99;
}

/* ===== Sidebar ===== */
.sidebar {
  position: absolute;
  top: 0; bottom: 0; left: 0;
  width: 280px;
  background: #fff;
  border-right: 1px solid #e5e5e5;
  z-index: 100;
  overflow-y: auto;
  -webkit-overflow-scrolling: touch;
}
.sidebar-header {
  padding: 20px 16px 12px;
  text-align: center;
}
.book-cover { margin-bottom: 12px; }
.book-cover img {
  max-width: 120px;
  max-height: 160px;
  border-radius: 4px;
}
.book-title {
  font-size: 15px;
  font-weight: 600;
  color: #222;
  margin-bottom: 4px;
}
.book-author { font-size: 13px; color: #888; margin-bottom: 4px; }
.book-format { font-size: 12px; color: #aaa; }
.sidebar-divider { height: 1px; background: #eee; margin: 8px 16px; }
.toc {
  list-style: none;
  padding: 4px 8px 16px;
  overflow-y: auto;
}
.toc li {
  padding: 10px 12px;
  cursor: pointer;
  font-size: 13px;
  color: #555;
  border-radius: 6px;
  margin-bottom: 2px;
  min-height: 44px;
  line-height: 24px;
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
}
.toc li:hover { background: #f5f5f5; }
.toc li.active { background: #eef4ff; color: #2563eb; font-weight: 500; }
.toc-num { display: inline-block; width: 28px; color: #bbb; font-size: 12px; }
.toc li.active .toc-num { color: #93b4f5; }

/* ===== Content ===== */
.content {
  position: absolute;
  top: 0; bottom: 0; left: 0; right: 0;
  overflow: hidden;
  background: #fafafa;
}
.reader-frame {
  width: 100%;
  height: 100%;
  border: none;
  background: #fff;
}

/* ===== Status pages ===== */
.status-page { padding: 80px 24px; text-align: center; }
.spinner {
  display: inline-block;
  width: 32px; height: 32px;
  border: 3px solid #e5e5e5;
  border-top-color: #2563eb;
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
  margin-bottom: 16px;
}
@keyframes spin { to { transform: rotate(360deg); } }
.status-text { font-size: 14px; color: #888; }
.error-icon { font-size: 36px; color: #e53e3e; margin-bottom: 12px; }
.error-text { color: #c53030; }
.btn-action {
  margin-top: 16px;
  padding: 10px 24px;
  background: #2563eb;
  color: #fff;
  border: none;
  border-radius: 6px;
  cursor: pointer;
  font-size: 14px;
  min-height: 44px;
}
.btn-action:hover { background: #1d4ed8; }
.welcome-icon { font-size: 48px; margin-bottom: 16px; }
.welcome-title { font-size: 22px; font-weight: 600; color: #222; margin-bottom: 8px; }
.welcome-desc { font-size: 14px; color: #888; margin-bottom: 16px; }
.welcome-example {
  background: #f0f0f0;
  border-radius: 6px;
  padding: 12px 16px;
  display: inline-block;
  margin-bottom: 16px;
}
.welcome-example code { font-size: 12px; color: #555; word-break: break-all; }
.welcome-formats { font-size: 12px; color: #aaa; }

/* ===== Footer ===== */
.footer {
  position: absolute;
  bottom: 0; left: 0; right: 0;
  height: 44px;
  background: #fff;
  border-top: 1px solid #e5e5e5;
  text-align: center;
  line-height: 44px;
  z-index: 50;
}
.btn-nav {
  background: none;
  border: 1px solid #ddd;
  border-radius: 4px;
  padding: 6px 20px;
  cursor: pointer;
  font-size: 16px;
  color: #555;
  min-height: 36px;
  vertical-align: middle;
}
.btn-nav:hover { background: #f5f5f5; }
.btn-nav:disabled { color: #ccc; border-color: #eee; cursor: default; }
.btn-nav:disabled:hover { background: none; }
.footer-info {
  display: inline-block;
  margin: 0 16px;
  font-size: 13px;
  color: #888;
  vertical-align: middle;
}

/* ===== Narrow screen: sidebar full width ===== */
@media (max-width: 40em) {
  .sidebar { width: 85%; }
  .header-title { display: none; }
  .btn-icon { min-width: 40px; min-height: 40px; margin-left: 2px; padding: 4px 6px; }
  .zoom-label { display: none; }
}
</style>
