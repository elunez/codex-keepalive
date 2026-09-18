package main

import (
	"html"
	"strings"
)

// 页面图标采用 Bootstrap Icons 的路径数据，并随插件页面内嵌，避免额外网络依赖。
const statusPageHTML = `<!doctype html>
<html lang="zh-CN">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>Codex 定时唤醒</title>
<style>
:root{color-scheme:light dark;--canvas:#f3f7fc;--surface:#fff;--surface-2:#f8f9fb;--ink:#1d2939;--muted:#667085;--line:#e4e7ec;--line-strong:#d0d5dd;--blue:#5276ad;--blue-soft:#eef3f9;--red:#cf4452;--red-soft:#fff1f2;--gray-soft:#eef1f5;--shadow:0 12px 34px rgba(16,24,40,.08)}
@media(prefers-color-scheme:dark){:root{--canvas:#11151c;--surface:#1a2029;--surface-2:#202630;--ink:#f1f4f8;--muted:#9aa5b5;--line:#303846;--line-strong:#414c5c;--blue:#7d9dcb;--blue-soft:#25344a;--red:#ef7380;--red-soft:#48282e;--gray-soft:#2a323e;--shadow:none}}
*{box-sizing:border-box}html,body{height:100%;min-height:100%;background:var(--canvas)}body{margin:0;min-height:100vh;color:var(--ink);font-family:-apple-system,BlinkMacSystemFont,"Segoe UI","PingFang SC","Noto Sans CJK SC",sans-serif}.shell{width:min(1400px,calc(100% - 48px));min-height:100vh;margin:0 auto;padding:24px 0 42px}.topbar{display:flex;align-items:center;justify-content:space-between;gap:20px;margin-bottom:20px}.brand{display:flex;align-items:center;gap:16px}.brand-mark{display:grid;place-items:center;width:62px;height:62px;border-radius:15px;background:var(--blue);color:#fff}.brand-mark svg{width:34px;height:34px}.brand-copy{min-width:0}.title-row{display:flex;align-items:center;gap:12px}.title-row h1{margin:0;font-size:27px;line-height:1.15;font-weight:760;letter-spacing:-.035em}.version{padding:5px 9px;border:1px solid color-mix(in srgb,var(--blue) 25%,var(--line));border-radius:8px;background:var(--blue-soft);color:var(--muted);font:12px/1 ui-monospace,SFMono-Regular,Menlo,monospace}.subtitle{margin:7px 0 0;color:var(--muted);font-size:14px}
.topbar-actions{display:flex;align-items:center;gap:10px}
.switch{position:relative;width:50px;height:28px;border-radius:999px;background:var(--line-strong);cursor:pointer;transition:.2s}.switch:after{content:"";position:absolute;left:3px;top:3px;width:22px;height:22px;border-radius:50%;background:#fff;box-shadow:0 1px 4px rgba(0,0,0,.18);transition:.2s}.switch.on{background:var(--blue)}.switch.on:after{transform:translateX(22px)}
.schedule{display:grid;grid-template-columns:repeat(4,minmax(0,1fr));align-items:center;min-height:76px;margin-bottom:22px;border:1px solid var(--line);border-radius:13px;background:var(--surface);box-shadow:var(--shadow);overflow:visible}.schedule-item{display:flex;align-items:center;justify-content:flex-start;gap:13px;min-width:0;padding:0 28px;border-right:1px solid var(--line)}.schedule-item:last-child{border-right:0}.schedule-item svg{flex:0 0 auto;width:21px;height:21px;color:var(--blue)}.schedule-text{display:flex;align-items:baseline;gap:10px;min-width:0;white-space:nowrap}.schedule-text span{color:var(--muted);font-size:13px}.schedule-text strong{overflow:hidden;text-overflow:ellipsis;font-size:15px}.icon-button{display:grid;place-items:center;width:42px;height:42px;padding:0;border:1px solid var(--line-strong);border-radius:10px;background:var(--surface);color:var(--ink);cursor:pointer;transition:border-color .16s,background .16s,transform .16s}.icon-button:hover{border-color:var(--blue);background:var(--surface);color:var(--blue)}.icon-button:active{transform:translateY(1px)}.icon-button svg{width:21px;height:21px}.icon-button:disabled{cursor:not-allowed;opacity:.55}.icon-button.is-loading svg{animation:spin .8s linear infinite}@keyframes spin{to{transform:rotate(360deg)}}
.logs-panel{border:1px solid var(--line);border-radius:14px;background:var(--surface);box-shadow:var(--shadow);overflow:hidden}.logs-head{display:flex;align-items:center;justify-content:space-between;gap:20px;padding:24px 22px}.logs-heading{display:flex;align-items:center;gap:12px;min-width:0}.logs-title{display:flex;align-items:center;gap:13px}.title-icon{display:grid;place-items:center;flex:0 0 42px;width:42px;height:42px;border-radius:10px;background:var(--blue-soft);color:var(--blue)}.title-icon svg{width:22px;height:22px}.logs-title h2{margin:0 0 4px;font-size:20px;letter-spacing:-.02em}.logs-title p{margin:0;color:var(--muted);font-size:12px}.filters{display:flex;align-items:center;justify-content:flex-end;gap:10px;min-width:0}.search{position:relative;width:200px}.search svg{position:absolute;left:13px;top:50%;width:17px;height:17px;color:var(--muted);transform:translateY(-50%);pointer-events:none}.search input{width:100%;padding-left:40px}.control,select,input{min-height:42px;border:1px solid var(--line-strong);border-radius:9px;background:var(--surface);color:var(--ink);font:inherit;font-size:12px}select{min-width:136px;padding:0 42px 0 13px;appearance:none;-webkit-appearance:none;background-color:var(--surface);background-image:url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='16' height='16' viewBox='0 0 24 24' fill='none' stroke='%23667085' stroke-width='2' stroke-linecap='round' stroke-linejoin='round'%3E%3Cpath d='m6 9 6 6 6-6'/%3E%3C/svg%3E");background-repeat:no-repeat;background-position:right 16px center;background-size:16px}input{padding:0 13px}.clear-button{display:flex;align-items:center;gap:8px;min-height:42px;padding:0 14px;border:1px solid var(--line-strong);border-radius:9px;background:var(--surface);color:var(--ink);font-size:12px;font-weight:650;cursor:pointer;white-space:nowrap}.clear-button:hover{border-color:var(--blue);background:var(--surface);color:var(--blue)}.clear-button svg{width:16px;height:16px}
.table-wrap{margin:0 22px;max-height:377px;overflow:auto;border:1px solid var(--line);border-radius:10px;scrollbar-gutter:stable}.logs-table{width:100%;min-width:980px;border-collapse:collapse;table-layout:fixed}.logs-table thead{position:sticky;top:0;z-index:1}.logs-table th{height:47px;padding:0 18px;background:var(--surface-2);color:var(--ink);font-size:12px;font-weight:720;text-align:left}.logs-table td{height:55px;padding:0 18px;border-top:1px solid var(--line);font-size:12px;vertical-align:middle}.logs-table th:nth-child(1){width:5%}.logs-table th:nth-child(2){width:17%}.logs-table th:nth-child(3){width:18%}.logs-table th:nth-child(4){width:10%}.logs-table th:nth-child(5){width:12%}.logs-table th:nth-child(6){width:9%}.logs-table th:nth-child(7){width:11%}.logs-table th:nth-child(8){width:18%}.index-cell{color:var(--muted);font-variant-numeric:tabular-nums}.account-cell,.detail-cell{overflow:hidden;text-overflow:ellipsis;white-space:nowrap}.tag,.result-pill{display:inline-flex;align-items:center;justify-content:center;min-height:26px;padding:4px 10px;border-radius:8px;font-size:11px;font-weight:680}.tag.manual{background:var(--blue-soft);color:var(--blue)}.tag.scheduled{background:var(--gray-soft);color:var(--muted)}.result-pill{gap:6px;border-radius:999px}.result-pill:before{content:"";width:7px;height:7px;border-radius:50%;background:currentColor}.result-pill.success{background:var(--blue-soft);color:var(--blue)}.result-pill.failed{background:var(--red-soft);color:var(--red)}.result-pill.skipped{background:var(--gray-soft);color:var(--muted)}.empty-row td{height:280px;text-align:center;color:var(--muted)}
.pager{display:flex;align-items:center;justify-content:space-between;gap:20px;min-height:86px;margin:18px 22px 0;border-top:1px solid var(--line)}.pager-summary,.pager-controls,.page-jump{display:flex;align-items:center;gap:10px}.pager-summary{color:var(--muted);font-size:12px}.pager-summary select{min-width:108px;min-height:38px;color:var(--ink)}.pager-controls{margin-left:auto}.page-button{display:grid;place-items:center;min-width:36px;height:36px;padding:0 8px;border:1px solid var(--line);border-radius:8px;background:var(--surface);color:var(--ink);font-size:12px;cursor:pointer}.page-button:hover:not(:disabled){border-color:var(--blue);color:var(--blue)}.page-button.current{border-color:var(--blue);background:var(--blue);color:#fff}.page-button:disabled{opacity:.4;cursor:not-allowed}.ellipsis{display:grid;place-items:center;width:24px;color:var(--muted)}.page-jump{color:var(--muted);font-size:12px}.page-jump input{width:56px;min-height:36px;text-align:center}.notice{position:fixed;right:20px;bottom:20px;z-index:120;max-width:min(400px,calc(100vw - 40px));padding:11px 14px;border-radius:9px;background:var(--ink);color:var(--surface);font-size:12px;box-shadow:var(--shadow)}.notice.error{background:var(--red);color:#fff}.auth-state{margin:60px 22px;padding:46px 20px;border:1px dashed var(--line-strong);border-radius:12px;text-align:center;color:var(--muted)}.auth-state strong{display:block;margin-bottom:7px;color:var(--ink)}.auth-state span{display:block;max-width:680px;margin:0 auto;line-height:1.7}button,input,select{outline:none}:focus-visible{outline:3px solid color-mix(in srgb,var(--blue) 28%,transparent);outline-offset:2px}
.refresh-button.is-loading svg{animation:spin .8s linear infinite}
.modal-backdrop{position:fixed;inset:0;z-index:100;display:grid;place-items:center;padding:20px;background:rgba(15,23,42,.45);backdrop-filter:blur(6px);-webkit-backdrop-filter:blur(6px);animation:fadeIn .18s ease-out}
.modal-backdrop[hidden]{display:none}
.modal-card{width:min(560px,100%);max-height:calc(100vh - 40px);overflow-y:auto;border:1px solid var(--line);border-radius:16px;background:var(--surface);box-shadow:0 24px 48px -12px rgba(0,0,0,.28);animation:scaleIn .2s cubic-bezier(.16,1,.3,1)}
.modal-head{display:flex;align-items:flex-start;justify-content:space-between;gap:16px;padding:22px 24px 18px;border-bottom:1px solid var(--line)}
.modal-title-box h3{margin:0 0 4px;font-size:18px;font-weight:750;letter-spacing:-.02em}
.modal-subtitle{margin:0;font-size:12px;color:var(--muted)}
.modal-close{display:grid;place-items:center;width:32px;height:32px;padding:0;border:none;border-radius:8px;background:transparent;color:var(--muted);font-size:22px;line-height:1;cursor:pointer;transition:background .15s,color .15s}
.modal-close:hover{background:var(--surface-2);color:var(--ink)}
.modal-form{display:flex;flex-direction:column;gap:16px;padding:22px 24px 24px}
.form-row{display:flex;flex-direction:column;gap:6px}
.form-switch-row{flex-direction:row;align-items:center;justify-content:space-between;padding:12px 16px;border:1px solid var(--line);border-radius:10px;background:var(--surface-2)}
.form-label-box{display:flex;flex-direction:column;gap:2px}
.form-label{font-size:13px;font-weight:680;color:var(--ink)}
.form-tip{font-size:11px;color:var(--muted)}
.form-grid-2{display:grid;grid-template-columns:1fr 1fr;gap:14px}
.preset-group{display:flex;align-items:center;gap:8px;margin-top:2px;flex-wrap:wrap}
.preset-btn{display:inline-flex;align-items:center;padding:4px 10px;border:1px dashed var(--line-strong);border-radius:6px;background:var(--surface);color:var(--muted);font-size:11px;cursor:pointer;transition:border-color .15s,color .15s,background .15s}
.preset-btn:hover{border-color:var(--blue);color:var(--blue);background:var(--surface)}
.modal-foot{display:flex;align-items:center;justify-content:flex-end;gap:10px;margin-top:4px;padding-top:16px;border-top:1px solid var(--line)}
.btn-cancel,.btn-save{display:inline-flex;align-items:center;justify-content:center;min-height:38px;padding:0 18px;border-radius:9px;font-size:13px;font-weight:680;cursor:pointer;transition:background .16s,border-color .16s,transform .16s}
.btn-cancel{border:1px solid var(--line-strong);background:var(--surface);color:var(--ink)}
.btn-cancel:hover{border-color:var(--blue);background:var(--surface);color:var(--blue)}
.btn-save{border:1px solid var(--blue);background:var(--blue);color:#fff}
.btn-save:hover:not(:disabled){background:color-mix(in srgb,var(--blue) 88%,#000);border-color:color-mix(in srgb,var(--blue) 88%,#000)}
.btn-save:active:not(:disabled){transform:translateY(1px)}
.btn-save:disabled{opacity:.55;cursor:not-allowed}
@keyframes fadeIn{from{opacity:0}to{opacity:1}}
@keyframes scaleIn{from{opacity:0;transform:scale(.96)}to{opacity:1;transform:scale(1)}}
@media(max-width:1000px){.shell{width:min(100% - 28px,1400px)}.schedule{grid-template-columns:repeat(2,1fr)}.schedule-item{min-height:62px}.schedule-item:nth-child(even){border-right:0}.logs-head{align-items:flex-start;flex-direction:column}.filters{width:100%;justify-content:flex-start;flex-wrap:wrap}.search{width:min(100%,360px)}.pager{flex-wrap:wrap}.pager-controls{margin-left:0}}
@media(max-width:620px){.shell{width:min(100% - 20px,1400px);padding-top:16px}.brand-mark{width:48px;height:48px;border-radius:12px}.brand-mark svg{width:28px;height:28px}.title-row h1{font-size:20px}.version,.subtitle{display:none}.schedule{grid-template-columns:1fr}.schedule-item{justify-content:flex-start;padding:0 18px;border-right:0;border-bottom:1px solid var(--line)}.schedule-item:last-child{border-bottom:0}.logs-head{padding:18px 14px}.logs-heading{width:auto;justify-content:flex-start}.filters{display:grid;grid-template-columns:1fr 1fr}.search{grid-column:1/-1;width:100%}.filters select{min-width:0;width:100%}.clear-button{justify-content:center}.table-wrap{margin:0 14px}.pager{align-items:flex-start;margin:14px;min-height:110px}.pager-summary{width:100%}.page-jump{display:none}.form-grid-2{grid-template-columns:1fr}}
@media(prefers-reduced-motion:reduce){*,*:before,*:after{transition:none!important;animation:none!important}}
</style>
</head>
<body>
<main class="shell">
  <header class="topbar">
    <div class="brand">
      <span class="brand-mark" aria-hidden="true"><svg viewBox="0 0 16 16" fill="currentColor"><path d="M8 3.5a4.5 4.5 0 1 0 4.5 4.5A4.505 4.505 0 0 0 8 3.5Zm0 8A3.5 3.5 0 1 1 11.5 8 3.504 3.504 0 0 1 8 11.5Z"/><path d="M7.5 5.5h1V8l2 1.2-.5.8-2.5-1.5V5.5ZM3.03 1.97l1.5 1.5-.7.7-1.5-1.5.7-.7Zm9.94 0 .7.7-1.5 1.5-.7-.7 1.5-1.5ZM6 0h4v1H6V0Z"/></svg></span>
      <div class="brand-copy"><div class="title-row"><h1>Codex 定时唤醒</h1><span class="version" id="version">v{{PLUGIN_VERSION}}</span></div><p class="subtitle">按计划定时唤醒 Codex，保持服务活跃</p></div>
    </div>
    <div class="topbar-actions">
      <button class="icon-button header-execute" id="execute" type="button" aria-label="立即执行" data-tooltip="立即执行" title="立即执行"><svg viewBox="0 0 16 16" fill="currentColor" aria-hidden="true"><path fill-rule="evenodd" clip-rule="evenodd" d="M8 3a5 5 0 1 0 4.546 2.914.5.5 0 0 1 .908-.418A6 6 0 1 1 8 2v1Z"/><path d="M8 4.466V.534a.25.25 0 0 1 .41-.192l2.36 1.966a.25.25 0 0 1 0 .384L8.41 4.658A.25.25 0 0 1 8 4.466Z"/></svg></button>
      <button class="icon-button header-settings" id="open-settings" type="button" aria-label="插件设置" data-tooltip="插件设置" title="插件设置"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M12 15a3 3 0 1 0 0-6 3 3 0 0 0 0 6Z"/><path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1Z"/></svg></button>
    </div>
  </header>
  <section class="schedule" aria-label="唤醒计划">
    <div class="schedule-item"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><rect x="3" y="4.5" width="18" height="16" rx="2"/><path d="M8 2.5v4M16 2.5v4M3 9.5h18"/></svg><div class="schedule-text"><span>下次执行</span><strong id="next-run">—</strong></div></div>
    <div class="schedule-item"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" aria-hidden="true"><circle cx="12" cy="12" r="9"/><path d="M3 12h18M12 3c2.6 2.4 3.8 5.4 3.8 9s-1.2 6.6-3.8 9M12 3c-2.6 2.4-3.8 5.4-3.8 9s1.2 6.6 3.8 9"/></svg><div class="schedule-text"><span>时区</span><strong id="timezone">—</strong></div></div>
    <div class="schedule-item"><svg viewBox="0 0 16 16" fill="currentColor" aria-hidden="true"><path d="M8 3a5 5 0 1 0 4.546 2.914.5.5 0 0 1 .908-.418A6 6 0 1 1 8 2v1Z"/><path d="M8 4.466V.534a.25.25 0 0 1 .41-.192l2.36 1.966a.25.25 0 0 1 0 .384L8.41 4.658A.25.25 0 0 1 8 4.466Z"/></svg><div class="schedule-text"><span>每账号</span><strong id="request-count">—</strong></div></div>
    <div class="schedule-item"><svg viewBox="0 0 16 16" fill="currentColor" aria-hidden="true"><path fill-rule="evenodd" d="M10.354 1.646a.5.5 0 0 1 0 .708L8.707 4H11.5A2.5 2.5 0 0 1 14 6.5v1a.5.5 0 0 1-1 0v-1A1.5 1.5 0 0 0 11.5 5H8.707l1.647 1.646a.5.5 0 0 1-.708.708l-2.5-2.5a.5.5 0 0 1 0-.708l2.5-2.5a.5.5 0 0 1 .708 0ZM5.646 8.646a.5.5 0 0 1 .708.708L4.707 11H7.5A1.5 1.5 0 0 0 9 9.5v-1a.5.5 0 0 1 1 0v1A2.5 2.5 0 0 1 7.5 12H4.707l1.647 1.646a.5.5 0 0 1-.708.708l-2.5-2.5a.5.5 0 0 1 0-.708l2.5-2.5Z"/></svg><div class="schedule-text"><span>随机延迟</span><strong id="random-delay">—</strong></div></div>
  </section>
  <section class="logs-panel">
    <header class="logs-head">
      <div class="logs-heading"><div class="logs-title"><span class="title-icon" aria-hidden="true"><svg viewBox="0 0 16 16" fill="currentColor"><path d="M14 4.5V14a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V2a2 2 0 0 1 2-2h5.5L14 4.5ZM9.5 1H4a1 1 0 0 0-1 1v12a1 1 0 0 0 1 1h8a1 1 0 0 0 1-1V4.5L9.5 1Z"/><path d="M5 7.5A.5.5 0 0 1 5.5 7h5a.5.5 0 0 1 0 1h-5a.5.5 0 0 1-.5-.5Zm0 3a.5.5 0 0 1 .5-.5h5a.5.5 0 0 1 0 1h-5a.5.5 0 0 1-.5-.5Z"/></svg></span><div><h2>执行日志</h2></div></div></div>
      <div class="filters">
        <label class="search"><svg viewBox="0 0 16 16" fill="currentColor" aria-hidden="true"><path d="M11.742 10.344a6.5 6.5 0 1 0-1.397 1.398h-.001c.03.04.062.078.098.115l3.85 3.85a1 1 0 0 0 1.415-1.414l-3.85-3.85a1.007 1.007 0 0 0-.115-.1ZM12 6.5a5.5 5.5 0 1 1-11 0 5.5 5.5 0 0 1 11 0Z"/></svg><input id="account-filter" type="search" placeholder="搜索账号" autocomplete="off"></label>
        <select id="result-filter" aria-label="结果筛选"><option value="all">全部结果</option><option value="success">成功</option><option value="failed">失败</option><option value="skipped">已跳过</option></select>
        <select id="trigger-filter" aria-label="触发方式筛选"><option value="all">全部方式</option><option value="scheduled">定时</option><option value="manual">手动</option></select>
        <button class="clear-button refresh-button" id="refresh-logs" type="button" aria-label="刷新列表" title="刷新列表"><svg viewBox="0 0 16 16" fill="currentColor" aria-hidden="true"><path fill-rule="evenodd" clip-rule="evenodd" d="M8 3a5 5 0 1 0 4.546 2.914.5.5 0 0 1 .908-.418A6 6 0 1 1 8 2v1Z"/><path d="M8 4.466V.534a.25.25 0 0 1 .41-.192l2.36 1.966a.25.25 0 0 1 0 .384L8.41 4.658A.25.25 0 0 1 8 4.466Z"/></svg><span>刷新列表</span></button>
        <button class="clear-button" id="clear-logs" type="button"><svg viewBox="0 0 16 16" fill="currentColor" aria-hidden="true"><path d="M5.5 5.5A.5.5 0 0 1 6 6v6a.5.5 0 0 1-1 0V6a.5.5 0 0 1 .5-.5Zm2.5 0a.5.5 0 0 1 .5.5v6a.5.5 0 0 1-1 0V6a.5.5 0 0 1 .5-.5Zm3 .5a.5.5 0 0 0-1 0v6a.5.5 0 0 0 1 0V6Z"/><path d="M14.5 3a1 1 0 0 1-1 1H13v9a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V4h-.5a1 1 0 0 1-1-1V2h4V1a1 1 0 0 1 1-1h3a1 1 0 0 1 1 1v1h4v1ZM4.118 4 4 4.059V13a1 1 0 0 0 1 1h6a1 1 0 0 0 1-1V4.059L11.882 4H4.118ZM6.5 1v1h3V1h-3Z"/></svg><span>清空日志</span></button>
      </div>
    </header>
    <div id="content"><div class="auth-state"><strong>正在连接</strong><span>读取定时唤醒状态…</span></div></div>
  </section>
</main>
<div class="modal-backdrop" id="settings-modal" hidden>
  <div class="modal-card" role="dialog" aria-modal="true" aria-labelledby="settings-title">
    <div class="modal-head">
      <div class="modal-title-box">
        <h3 id="settings-title">插件设置</h3>
        <p class="modal-subtitle">配置定时唤醒执行计划、模型与请求参数</p>
      </div>
      <button class="modal-close" id="close-settings" type="button" aria-label="关闭">&times;</button>
    </div>
    <form id="settings-form" class="modal-form">
      <div class="form-row form-switch-row">
        <div class="form-label-box">
          <label class="form-label">定时自动执行</label>
          <span class="form-tip">按预设时间计划自动发起唤醒请求；关闭后仍支持手动执行</span>
        </div>
        <div class="switch on" id="cfg-enabled" role="switch" aria-checked="true" tabindex="0"></div>
      </div>

      <div class="form-row">
        <div class="form-label-box">
          <label class="form-label" for="cfg-times">每日执行时间</label>
          <span class="form-tip">多个时间以英文逗号分隔（24小时制 HH:mm）</span>
        </div>
        <input class="control" id="cfg-times" type="text" placeholder="07:00,12:15,17:30" autocomplete="off" required>
        <div class="preset-group">
          <button type="button" class="preset-btn" data-preset="07:00,12:15,17:30">三频 (07:00,12:15,17:30)</button>
          <button type="button" class="preset-btn" data-preset="07:00,12:15,17:30,23:45">四频 (07:00,12:15,17:30,23:45)</button>
        </div>
      </div>

      <div class="form-grid-2">
        <div class="form-row">
          <label class="form-label" for="cfg-timezone">执行时区</label>
          <input class="control" id="cfg-timezone" type="text" placeholder="Asia/Shanghai" autocomplete="off" required>
        </div>
        <div class="form-row">
          <label class="form-label" for="cfg-model">唤醒模型</label>
          <input class="control" id="cfg-model" type="text" placeholder="gpt-5.6-sol" autocomplete="off" required>
        </div>
      </div>

      <div class="form-grid-2">
        <div class="form-row">
          <label class="form-label" for="cfg-requests">每账号请求次数 (1~10)</label>
          <input class="control" id="cfg-requests" type="number" min="1" max="10" placeholder="2" required>
        </div>
        <div class="form-row">
          <label class="form-label" for="cfg-concurrency">最大并发数 (1~32)</label>
          <input class="control" id="cfg-concurrency" type="number" min="1" max="32" placeholder="2" required>
        </div>
      </div>

      <div class="form-row">
        <div class="form-label-box">
          <label class="form-label" for="cfg-delay">随机错峰延迟上限 (秒)</label>
          <span class="form-tip">0~3600 秒，在此范围内为请求生成随机延迟（0 为不延迟）</span>
        </div>
        <input class="control" id="cfg-delay" type="number" min="0" max="3600" placeholder="60" required>
      </div>

      <div class="modal-foot">
        <button class="btn-cancel" id="cancel-settings" type="button">取消</button>
        <button class="btn-save" id="save-settings" type="submit">保存设置</button>
      </div>
    </form>
  </div>
</div>
<div class="notice" id="notice" hidden></div>
<script>
const API='/v0/management/plugins/codex-keepalive';
const STORAGE_PREFIX_V1='enc::v1::';
const STORAGE_PREFIX_V2='enc::v2::';
const STORAGE_SALT='cli-proxy-api-webui::secure-storage';
let managementKey='',page=1,pageSize=10,lastPage=null,statusTimer=null,searchTimer=null,loadingLogs=false;
const $=id=>document.getElementById(id);
function storageKeyBytes(version){const suffix=version==='v2'?('|v2|'+location.host):('|'+location.host+'|'+navigator.userAgent);return new TextEncoder().encode(STORAGE_SALT+suffix)}
function decodeStored(raw){if(!raw)return null;let text=raw;let version='',prefix='';if(raw.startsWith(STORAGE_PREFIX_V2)){version='v2';prefix=STORAGE_PREFIX_V2}else if(raw.startsWith(STORAGE_PREFIX_V1)){version='v1';prefix=STORAGE_PREFIX_V1}if(prefix){try{const encoded=atob(raw.slice(prefix.length)),bytes=Uint8Array.from(encoded,c=>c.charCodeAt(0)),key=storageKeyBytes(version);for(let i=0;i<bytes.length;i++)bytes[i]^=key[i%key.length];text=new TextDecoder().decode(bytes)}catch{return null}}try{return JSON.parse(text)}catch{return text}}
function savedManagementKey(){try{const direct=decodeStored(localStorage.getItem('managementKey'));if(typeof direct==='string'&&direct.trim())return direct.trim();const auth=decodeStored(localStorage.getItem('cli-proxy-auth')),key=auth&&auth.state&&auth.state.managementKey;return typeof key==='string'?key.trim():''}catch{return ''}}
async function request(path,options={}){if(!managementKey){renderAuthState();const error=new Error('未读取到管理凭证');error.status=401;throw error}const headers={'Authorization':'Bearer '+managementKey};if(options.body&&!options.headers)headers['Content-Type']='application/json;charset=utf-8';const response=await fetch(API+path,{...options,headers});let data={};try{data=await response.json()}catch{}if(!response.ok){const error=new Error(data.error||('请求失败：HTTP '+response.status));error.status=response.status;if(response.status===401||response.status===403)renderAuthState('管理凭证无效或已过期');throw error}return data}
function safe(value){return String(value??'').replace(/[&<>"]/g,c=>({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;'}[c]))}
function notice(text,error=false){const node=$('notice');node.textContent=text;node.className='notice'+(error?' error':'');node.hidden=false;clearTimeout(notice.timer);notice.timer=setTimeout(()=>node.hidden=true,3200)}
function renderAuthState(title='未读取到管理凭证',detail='请返回当前管理中心的登录页，重新登录并勾选“记住凭证”，然后刷新本页面。'){$('content').innerHTML='<div class="auth-state"><strong>'+safe(title)+'</strong><span>'+safe(detail)+'</span></div>'}
function validDate(value){if(!value)return null;const date=new Date(value);return Number.isNaN(date.getTime())||date.getUTCFullYear()<=1?null:date}
function dateTime(value){const date=validDate(value);return date?date.toLocaleString('zh-CN',{year:'numeric',month:'2-digit',day:'2-digit',hour:'2-digit',minute:'2-digit',second:'2-digit',hour12:false}).replaceAll('/','-'):'—'}
function nextText(value,enabled){if(!enabled)return '已停用';const date=validDate(value);if(!date)return '—';const now=new Date(),tomorrow=new Date(now);tomorrow.setDate(now.getDate()+1);const time=date.toLocaleTimeString('zh-CN',{hour:'2-digit',minute:'2-digit',hour12:false});if(date.toDateString()===now.toDateString())return '今天 '+time;if(date.toDateString()===tomorrow.toDateString())return '明天 '+time;return date.toLocaleDateString('zh-CN',{month:'2-digit',day:'2-digit'})+' '+time}
function renderStatus(data){$('version').textContent='v'+data.version;const enabled=!!data.enabled;$('next-run').textContent=nextText(data.next_scheduled_at,enabled);$('timezone').textContent=data.timezone||'—';$('request-count').textContent=Number(data.requests_per_run||0)+' 次';$('random-delay').textContent='0～'+Number(data.random_delay_max_seconds||0)+' 秒';const running=!!(data.run&&data.run.running),button=$('execute');button.disabled=running;button.classList.toggle('is-loading',running);button.setAttribute('aria-busy',String(running));button.setAttribute('aria-label',running?'执行中':'立即执行');clearTimeout(statusTimer);statusTimer=setTimeout(refreshStatus,running?1500:15000)}
async function refreshStatus(){try{const data=await request('/status');renderStatus(data);await loadLogs(false)}catch(error){notice(error.message,true)}}
function queryString(){const query=new URLSearchParams({page:String(page),page_size:String(pageSize),result:$('result-filter').value,trigger:$('trigger-filter').value});const account=$('account-filter').value.trim();if(account)query.set('account',account);return query.toString()}
function triggerTag(value){if(value==='manual')return '<span class="tag manual">手动</span>';return '<span class="tag scheduled">定时</span>'}
function progressText(item){return item.request_index>0?('第 '+item.request_index+'/'+item.request_total+' 次'):'—'}
function resultPill(status){const labels={success:'成功',failed:'失败',skipped:'已跳过'};const normalized=labels[status]?status:'skipped';return '<span class="result-pill '+normalized+'">'+labels[normalized]+'</span>'}
function renderRows(items,offset){if(!items.length)return '<tr class="empty-row"><td colspan="8">暂无符合条件的执行日志</td></tr>';return items.map((item,index)=>'<tr><td class="index-cell">'+(offset+index+1)+'</td><td>'+safe(dateTime(item.executed_at))+'</td><td class="account-cell" title="'+safe(item.account)+'">'+safe(item.account)+'</td><td>'+triggerTag(item.trigger)+'</td><td>'+safe(progressText(item))+'</td><td>'+(item.request_index>0?safe(item.delay_seconds)+' 秒':'—')+'</td><td>'+resultPill(item.status)+'</td><td class="detail-cell" title="'+safe(item.detail)+'">'+safe(item.detail)+'</td></tr>').join('')}
function visiblePages(current,total){if(total<=7)return Array.from({length:total},(_,i)=>i+1);const result=[1];const start=Math.max(2,current-1),end=Math.min(total-1,current+1);if(start>2)result.push('…');for(let value=start;value<=end;value++)result.push(value);if(end<total-1)result.push('…');result.push(total);return result}
function pagerButtons(data){const pages=visiblePages(data.page,data.total_pages);return '<button class="page-button" data-page="'+(data.page-1)+'" '+(data.page<=1?'disabled':'')+' aria-label="上一页">‹</button>'+pages.map(value=>value==='…'?'<span class="ellipsis">…</span>':'<button class="page-button '+(value===data.page?'current':'')+'" data-page="'+value+'">'+value+'</button>').join('')+'<button class="page-button" data-page="'+(data.page+1)+'" '+(data.page>=data.total_pages?'disabled':'')+' aria-label="下一页">›</button>'}
function renderLogs(data){lastPage=data;page=data.page;$('content').innerHTML='<div class="table-wrap"><table class="logs-table"><thead><tr><th>序号</th><th>执行时间</th><th>账号</th><th>触发方式</th><th>请求进度</th><th>随机延迟</th><th>结果</th><th>详情</th></tr></thead><tbody>'+renderRows(data.items||[],(data.page-1)*data.page_size)+'</tbody></table></div><footer class="pager"><div class="pager-summary"><span>共 '+data.total+' 条记录</span><select id="page-size" aria-label="每页条数"><option value="10" '+(data.page_size===10?'selected':'')+'>10 条/页</option><option value="20" '+(data.page_size===20?'selected':'')+'>20 条/页</option><option value="50" '+(data.page_size===50?'selected':'')+'>50 条/页</option><option value="100" '+(data.page_size===100?'selected':'')+'>100 条/页</option></select></div><nav class="pager-controls" aria-label="日志分页">'+pagerButtons(data)+'</nav><label class="page-jump">前往 <input id="page-jump" type="number" min="1" max="'+Math.max(1,data.total_pages)+'" value="'+data.page+'"> 页</label></footer>'}
async function loadLogs(showError=true){if(loadingLogs)return false;loadingLogs=true;try{const data=await request('/logs?'+queryString());renderLogs(data);return true}catch(error){if(showError)notice(error.message,true);return false}finally{loadingLogs=false}}
async function executeNow(){const button=$('execute');if(button.disabled)return;button.disabled=true;button.classList.add('is-loading');try{await request('/execute',{method:'POST'});notice('唤醒任务已开始');await refreshStatus();await loadLogs(false)}catch(error){notice(error.message,true);button.disabled=false;button.classList.remove('is-loading')}}
async function refreshLogs(){const button=$('refresh-logs');if(button.disabled)return;button.disabled=true;button.classList.add('is-loading');try{const data=await request('/status');renderStatus(data);if(await loadLogs())notice('执行日志已刷新')}catch(error){notice(error.message,true)}finally{button.disabled=false;button.classList.remove('is-loading')}}
async function clearLogs(){if(!confirm('确定清空全部执行日志吗？此操作无法撤销。'))return;try{await request('/logs',{method:'DELETE'});page=1;await loadLogs();notice('执行日志已清空')}catch(error){notice(error.message,true)}}
function resetAndLoad(){page=1;loadLogs()}

async function openSettingsModal(){
  const btn=$('open-settings');
  if(!managementKey){renderAuthState();return}
  btn.disabled=true;
  try{
    // 使用专用配置接口读取 config.json，字段与保存接口保持一致，
    // 避免把 /status 的展示字段误当成配置字段而回填默认值。
    const data=await request('/settings');
    const enabled=data.activation_enabled!==false;
    $('cfg-enabled').classList.toggle('on',enabled);
    $('cfg-enabled').setAttribute('aria-checked',String(enabled));
    $('cfg-times').value=data.activation_times||'';
    $('cfg-timezone').value=data.activation_timezone||'Asia/Shanghai';
    $('cfg-model').value=data.activation_model||'gpt-5.6-sol';
    $('cfg-requests').value=data.activation_requests_per_run||2;
    $('cfg-concurrency').value=data.activation_concurrency||2;
    $('cfg-delay').value=data.activation_random_delay_seconds!=null?data.activation_random_delay_seconds:60;
    $('settings-modal').hidden=false;
  }catch(err){
    notice('读取配置失败: '+err.message,true);
  }finally{
    btn.disabled=false;
  }
}
function closeSettingsModal(){
  $('settings-modal').hidden=true;
}

async function initialize(){managementKey=savedManagementKey();if(!managementKey){renderAuthState();return}try{const data=await request('/status');renderStatus(data);await loadLogs()}catch(error){if(error.status!==401&&error.status!==403){$('content').innerHTML='<div class="auth-state"><strong>无法读取插件状态</strong><span>'+safe(error.message)+'</span></div>';notice(error.message,true)}}}

$('execute').addEventListener('click',executeNow);
$('open-settings').addEventListener('click',openSettingsModal);
$('close-settings').addEventListener('click',closeSettingsModal);
$('cancel-settings').addEventListener('click',closeSettingsModal);
$('settings-modal').addEventListener('click',e=>{if(e.target===$('settings-modal'))closeSettingsModal()});
document.addEventListener('keydown',e=>{if(e.key==='Escape'&&!$('settings-modal').hidden)closeSettingsModal()});
$('cfg-enabled').addEventListener('click',()=>{
  const next=!$('cfg-enabled').classList.contains('on');
  $('cfg-enabled').classList.toggle('on',next);
  $('cfg-enabled').setAttribute('aria-checked',String(next));
});
document.querySelectorAll('.preset-btn').forEach(btn=>{
  btn.addEventListener('click',()=>{
    $('cfg-times').value=btn.dataset.preset;
  });
});
$('settings-form').addEventListener('submit',async e=>{
  e.preventDefault();
  const saveBtn=$('save-settings');
  if(saveBtn.disabled)return;
  saveBtn.disabled=true;
  saveBtn.textContent='保存中…';
  const payload={
    activation_enabled:$('cfg-enabled').classList.contains('on'),
    activation_times:$('cfg-times').value.trim(),
    activation_timezone:$('cfg-timezone').value.trim(),
    activation_model:$('cfg-model').value.trim(),
    activation_requests_per_run:Number($('cfg-requests').value)||2,
    activation_concurrency:Number($('cfg-concurrency').value)||2,
    activation_random_delay_seconds:Number($('cfg-delay').value)||0
  };
  try{
    await request('/settings',{method:'PUT',body:JSON.stringify(payload)});
    notice('配置保存成功');
    closeSettingsModal();
    await refreshStatus();
  }catch(err){
    notice(err.message,true);
  }finally{
    saveBtn.disabled=false;
    saveBtn.textContent='保存设置';
  }
});
$('refresh-logs').addEventListener('click',refreshLogs);
$('clear-logs').addEventListener('click',clearLogs);
$('result-filter').addEventListener('change',resetAndLoad);
$('trigger-filter').addEventListener('change',resetAndLoad);
$('account-filter').addEventListener('input',()=>{clearTimeout(searchTimer);searchTimer=setTimeout(resetAndLoad,300)});
$('content').addEventListener('click',event=>{const button=event.target.closest('[data-page]');if(!button||button.disabled)return;page=Number(button.dataset.page);loadLogs()});
$('content').addEventListener('change',event=>{if(event.target.id==='page-size'){pageSize=Number(event.target.value);page=1;loadLogs()}if(event.target.id==='page-jump'){const target=Math.max(1,Math.min(Number(event.target.value)||1,lastPage&&lastPage.total_pages||1));page=target;loadLogs()}});
initialize();
</script>
</body>
</html>`

func renderStatusPage() string {
	return strings.Replace(statusPageHTML, "{{PLUGIN_VERSION}}", html.EscapeString(pluginVersion), 1)
}
