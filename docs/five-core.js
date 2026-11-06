"use strict";
const NAV=[['ai','AI Investment Chatbot','✧'],['personal','Personalised Intelligence','◉'],['automation','Continuous Automation','◷'],['community','Community Integration','◎'],['humans','Investor Digital Humans','◇']];
const TICKERS=[{s:'NVDA',n:'NVIDIA Corporation',p:125.60,c:1.72},{s:'AAPL',n:'Apple Inc.',p:215.40,c:-0.43},{s:'MSFT',n:'Microsoft Corporation',p:390.80,c:0.88},{s:'SPY',n:'SPDR S&P 500 ETF',p:512.30,c:0.57},{s:'GOOGL',n:'Alphabet Inc.',p:174.20,c:-0.24}];
const byId=id=>document.getElementById(id);
const usd=v=>'$'+Number(v).toLocaleString('en-US',{minimumFractionDigits:2,maximumFractionDigits:2});
const esc=v=>String(v).replace(/[&<>"']/g,c=>({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c]));
let tab='ai',cash=46280,holdings={NVDA:80,AAPL:55,MSFT:22},rules=[],messages=[],factor='balanced',selected='NVDA',tourMode=false;
let profile={risk:'Moderate',horizon:'Long term',sector:'Technology',experience:'Intermediate'};
let investor='Buffett';
const total=()=>cash+TICKERS.reduce((n,x)=>n+(holdings[x.s]||0)*x.p,0);
const pageHero=(title,desc)=>`<div class="head-row"><div><div class="eyebrow">INVESTMENT INTELLIGENCE</div><h1 class="page-title">${title}</h1><p class="page-sub">${desc}</p></div><div class="quick-actions"><button class="button button-sm" data-video="1">▶ Watch film</button></div></div>`;
const stat=(title,value,foot,cls='')=>`<div class="panel stat"><div class="label">${title}</div><strong class="${cls}">${value}</strong><small>${foot}</small></div>`;
 const pts=Array.from({length:60},(_,i)=>{let v=170-i*1.8+Math.sin(i*.44+seed)*14+Math.sin(i*.18+seed)*11;return {x:20+i*12.45,y:v}});
 let draw='';if(!candles){draw=`<defs><linearGradient id="shade-${seed}" x1="0" x2="0" y1="0" y2="1"><stop offset="0%" stop-color="#E7FFF5" stop-opacity=".75"/><stop offset="100%" stop-color="white" stop-opacity="0"/></linearGradient></defs><path d="M${pts.map(p=>p.x+','+p.y).join(' L')} L754,215 L20,215 Z" fill="url(#shade-${seed})"/><path d="M${pts.map(p=>p.x+','+p.y).join(' L')}" stroke="#21D59A" stroke-width="2.7" fill="none" stroke-linejoin="round"/>`}
 else {draw=pts.filter((_,i)=>i%2===0).map((p,i)=>{let o=p.y+Math.sin(i*1.7)*9,c=p.y+Math.sin(i*.98+1)*10,up=c<o;return `<line x1="${p.x}" x2="${p.x}" y1="${Math.min(o,c)-8}" y2="${Math.max(o,c)+8}" stroke="${up?'#21D59A':'#101010'}"/><rect x="${p.x-3.5}" y="${Math.min(o,c)}" width="7" height="${Math.max(2,Math.abs(o-c))}" rx="1" fill="${up?'#21D59A':'#101010'}"/>`}).join('')}
 let bars=Array.from({length:40},(_,i)=>`<rect x="${20+i*18}" y="${231-(8+i%7*2.4)}" width="7" height="${8+i%7*2.4}" rx="1" fill="${i%4===0?'#21D59A':'#EAEAEA'}"/>`).join('');return `<svg viewBox="0 0 800 260" role="img" aria-label="Illustrative synthetic chart, not real market prices"><g>${grid}</g>${draw}${bars}<g fill="#101010" font-family="Arial" font-size="12"><text x="20" y="257">Jan</text><text x="185" y="257">Mar</text><text x="362" y="257">Jun</text><text x="537" y="257">Sep</text><text x="710" y="257">Dec</text></g></svg>`}


function actionButton(label,id){return '<button type="button" class="button button-sm" data-go="'+id+'">'+label+'</button>'}
function researchPill(name,val){return '<span class="context-token"><b>'+name+'</b> '+esc(val)+'</span>'}
function capitalRows(){return Object.entries(holdings).filter(x=>x[1]>0).map(([t,qty])=>'<div class="data-line"><strong>'+t+'</strong><span>'+qty+' paper shares · '+usd(qty*(TICKERS.find(x=>x.s===t)?.p||0))+'</span></div>').join('')}
function scriptedAnswer(question){
 const technologyExposure=Object.entries(holdings).reduce((sum,[t,n])=>sum+(['NVDA','AAPL','MSFT','GOOGL'].includes(t)?n*(TICKERS.find(x=>x.s===t)?.p||0):0),0);
 const concentration=Math.round(100*technologyExposure/total());
 const q=question.toLowerCase();
 const mode=q.includes('buffett')?'Buffett':q.includes('graham')?'Graham':q.includes('munger')?'Munger':investor;
 const lens=mode==='Graham'?'a conservative valuation and margin-of-safety lens':mode==='Munger'?'an incentives, opportunity-cost, and inversion lens':'a business-quality, competitive-moat, and cash-generation lens';
 if(q.includes('alert')||q.includes('monitor'))return 'SIMULATED RESEARCH: There are '+rules.length+' example alert rules. Review price thresholds alongside current portfolio exposure and only update your thesis when verified evidence changes. The public demo does not monitor markets.';
 if(q.includes('community')||q.includes('sentiment'))return 'SIMULATED RESEARCH: Compare the example bullish thesis (AI demand) with the bearish thesis (competition and valuation expectations). These viewpoints are illustrative; no forum posts have been retrieved. Verify interpretations against original filings.';
 return 'SIMULATED AI RESPONSE — not a live model call. For '+profile.horizon.toLowerCase()+' research with '+profile.risk.toLowerCase()+' risk preference, start with dated financial statements, business risks, valuation and sector exposure. Your illustrative technology holdings account for roughly '+concentration+'% of paper equity at the fixed demo reference prices. '+mode+'-inspired analysis adds '+lens+'. Confirm source dates and assumptions before any real decision.';
}
function ai(){
 return '<div class="five-hero"><span class="eyebrow">01 · CORE INTELLIGENCE</span><h1>Research starts with a conversation.</h1><p>Ask a company question once. InvestOne brings your portfolio context, quantitative tools, community perspectives and investment frameworks together.</p><div class="hero-buttons">'+actionButton('Personalise my research →','personal')+actionButton('Review community viewpoints →','community')+'</div></div>'
 + '<div class="context-strip">'+researchPill('Risk',profile.risk)+researchPill('Horizon',profile.horizon)+researchPill('Interest',profile.sector)+researchPill('Paper equity',usd(total()))+'</div>'
 + '<div class="columns"><div class="panel"><div class="section-top"><h2>AI Investment Chatbot</h2><span class="badge badge-demo">SCRIPTED INTERACTIVE DEMO</span></div><div class="ai-bubble"><small>RESEARCH ASSISTANT</small><p>What would you like to investigate today? Explore a company, compare investment frameworks, or consider your existing portfolio.</p></div>'
 + '<div id="messages">'+messages.map(m=>'<div class="ai-bubble '+(m.user?'user':'')+'"><small>'+(m.user?'YOUR QUESTION':'SCRIPTED RESPONSE')+'</small><p>'+esc(m.message)+'</p></div>').join('')+'</div>'
 + '<form id="ask-form" class="fields" style="margin-top:16px"><label class="field" style="flex:1"><span>YOUR INVESTMENT QUESTION</span><input name="question" value="Should I add more NVIDIA to my portfolio?" maxlength="300" required></label><button type="submit" class="button button-primary">Ask →</button></form>'
 + '<div class="quick-prompts"><button data-prompt="How would Buffett assess NVIDIA given my portfolio?">Buffett perspective</button><button data-prompt="Compare NVIDIA community views and valuation risks">Compare opinions</button><button data-prompt="Which alerts should I create for NVDA?">Create monitoring plan</button></div>'
 + '<p class="watchlist-note">This public site uses scripted, context-sensitive examples, not a model API or real financial advice.</p></div>'
 + '<div class="panel"><div class="section-top"><h2>Connected research workflow</h2><span class="pill">ONE WORKSPACE</span></div>'
 + '<div class="feature-step"><span>01</span><div><b>Personal context</b><p>Risk tolerance, horizon and portfolio holdings guide analysis.</p>'+actionButton('Configure →','personal')+'</div></div>'
 + '<div class="feature-step"><span>02</span><div><b>Evidence and sentiment</b><p>Review example bullish and bearish viewpoints separately from filings.</p>'+actionButton('Explore →','community')+'</div></div>'
 + '<div class="feature-step"><span>03</span><div><b>Investor frameworks</b><p>Ask a question through multiple documented investment lenses.</p>'+actionButton('Compare →','humans')+'</div></div>'
 + '<div class="feature-step"><span>04</span><div><b>Follow-up monitoring</b><p>Build an example rule for a company you are researching.</p>'+actionButton('Configure →','automation')+'</div></div></div></div>';
}
function personal(){
 const opts=(name,list)=>list.map(x=>'<option value="'+x+'" '+(profile[name]===x?'selected':'')+'>'+x+'</option>').join('');
 const techPct=Math.round(100*Object.entries(holdings).reduce((sum,[t,n])=>sum+(['NVDA','AAPL','MSFT','GOOGL'].includes(t)?n*(TICKERS.find(x=>x.s===t)?.p||0):0),0)/total());
 return '<div class="five-hero"><span class="eyebrow">02 · PERSONALISED INVESTMENT INTELLIGENCE</span><h1>Market insights, made relevant to you.</h1><p>A company can mean something different to each investor. Connect personal preferences and paper holdings to every research question.</p>'+actionButton('Ask with this context →','ai')+'</div>'
 + '<div class="stat-grid">'+stat('Paper portfolio',usd(total()),'Synthetic reference prices')+stat('Technology exposure',techPct+'%','Illustrative equity exposure')+stat('Positions',Object.values(holdings).filter(n=>n>0).length,'Paper holdings')+stat('Risk profile',esc(profile.risk),'Demo preference')+'</div>'
 + '<div class="columns"><div class="panel"><div class="section-top"><h2>Personal research profile</h2><span class="badge badge-demo">SESSION-ONLY DEMO</span></div><form id="profile-form" class="profile-form">'
 + '<label class="field"><span>RISK TOLERANCE</span><select name="risk">'+opts('risk',['Conservative','Moderate','Growth-oriented'])+'</select></label>'
 + '<label class="field"><span>INVESTMENT HORIZON</span><select name="horizon">'+opts('horizon',['Short term','Medium term','Long term'])+'</select></label>'
 + '<label class="field"><span>SECTOR OF INTEREST</span><select name="sector">'+opts('sector',['Technology','Manufacturing','Healthcare','Diversified'])+'</select></label>'
 + '<label class="field"><span>FINANCIAL EXPERIENCE</span><select name="experience">'+opts('experience',['Beginner','Intermediate','Advanced'])+'</select></label>'
 + '<button class="button button-primary" type="submit">Save demo preferences</button><p id="profile-feedback" class="form-feedback">Selections affect the next scripted AI answer; no account is created.</p></form></div>'
 + '<div class="panel"><div class="section-top"><h2>Your portfolio context</h2>'</div>'+capitalRows()
 + '<div class="insight"><b>Why personalisation?</b>The same NVIDIA announcement matters differently when technology is already a significant share of your holdings. Context changes which risks deserve attention.</div>'
 + '<button type="button" class="button button-sm" data-prompt="Assess NVIDIA given my current holdings and risk profile">Analyse my exposure in AI →</button></div></div>';
}
function automation(){
 return '<div class="five-hero"><span class="eyebrow">03 · CONTINUOUS MONITORING & AUTOMATION</span><h1>From one-time research to consistent follow-up.</h1><p>Markets keep moving when you are away. Define the questions and thresholds you want the platform to revisit.</p>'+actionButton('Research an asset first →','ai')+'</div>'
 + '<div class="columns"><div class="panel"><div class="section-top"><h2>Create a monitoring rule</h2><span class="badge badge-demo">SIMULATED RULE</span></div>'
 + '<form id="rule-form" class="fields"><label class="field"><span>ASSET</span><select name="symbol">'+TICKERS.map(t=>'<option value="'+t.s+'">'+t.s+'</option>').join('')+'</select></label><label class="field"><span>TRIGGER</span><select name="direction"><option value="below">Price below</option><option value="above">Price above</option></select></label><label class="field"><span>THRESHOLD (USD)</span><input name="threshold" type="number" value="120" min=".01" step=".01" required></label><button class="button button-primary" type="submit">Add alert</button></form>'
 + '<p id="rule-feedback" class="form-feedback">Create a browser-only example rule. No live market checks or notifications.</p><div class="divider"></div>'
 + '<div class="section-top"><h2>Configured demo alerts</h2><span class="pill">'+rules.length+' RULES</span></div>'
 + (rules.length?rules.map(x=>'<div class="text-box"><b>'+esc(x)+'</b><p>Local session · example threshold only</p></div>').join(''):'<p class="page-sub">No rules yet. Add one to see it appear here.</p>')
 + '</div><div class="panel"><div class="section-top"><h2>Intended automation loop</h2><span class="pill">DESIGNED WORKFLOW</span></div>'
 + '<div class="feature-step"><span>1</span><div><b>Define</b><p>Set an asset, a threshold, or a future earnings-review question.</p></div></div>'
 + '<div class="feature-step"><span>2</span><div><b>Observe</b><p>The desktop backend can monitor selected data while it is running. This public preview does not poll providers.</p></div></div>'
 + '<div class="feature-step"><span>3</span><div><b>Reassess</b><p>When new evidence matters, return to personal context and AI research.</p>'+actionButton('Back to AI research →','ai')+'</div></div></div></div>';
}
function community(){
 return '<div class="five-hero"><span class="eyebrow">04 · FORUM & COMMUNITY INTEGRATION</span><h1>Understand both sides of the conversation.</h1><p>Public investment discussions can raise useful questions, but they are not the same as verified financial statements.</p>'+actionButton('Discuss these views with AI →','ai')+'</div>'
 + '<div class="side-by-side"><div class="panel"><div class="section-top"><h2>Potential bullish thesis</h2><span class="pill">ILLUSTRATIVE OPINION</span></div><div class="text-box"><b>Industry growth and platform advantages</b><p>One possible view: sustained accelerated-computing demand could support revenue growth and a strong ecosystem.</p></div><button class="button button-sm" data-prompt="Evaluate the bullish case for NVIDIA using evidence and my portfolio context">Evaluate bullish case →</button></div>'
 + '<div class="panel"><div class="section-top"><h2>Potential bearish thesis</h2><span class="pill">ILLUSTRATIVE OPINION</span></div><div class="text-box"><b>Valuation, competition, and concentration</b><p>Another possible view: elevated expectations, competition and customer concentration could increase downside risk.</p></div><button class="button button-sm" data-prompt="Evaluate the bearish case for NVIDIA using evidence and my portfolio context">Evaluate bearish case →</button></div></div>'
 + '<div class="panel section-gap"><div class="section-top"><h2>How community insights fit the platform</h2><span class="badge badge-demo">NO LIVE FORUM INGESTION</span></div>'
 + '<div class="workflow-line"><b>Public viewpoints</b><span>→</span><b>Separate claims from evidence</b><span>→</span><b>Cross-check with AI research</b><span>→</span><b>Adjust monitoring questions</b></div>'
 + '<p class="watchlist-note">No Reddit, Xueqiu or Eastmoney posts were collected. These are demonstration viewpoints, not measured sentiment.</p></div>';
}
function humans(){
 const descriptions={Buffett:['Business quality & cash flows','Consider competitive advantages, owner earnings and how much confidence the evidence supports.'],Munger:['Mental models & inversion','Ask what could go wrong, examine incentives and opportunity cost, and challenge easy assumptions.'],Graham:['Valuation & margin of safety','Pay attention to conservative earnings estimates, balance-sheet strength and downside protection.']};
 let current=descriptions[investor];
 return '<div class="five-hero"><span class="eyebrow">05 · INVESTOR DIGITAL HUMANS</span><h1>One question. Different investment lenses.</h1><p>Compare educational interpretations of publicly documented investment principles, then take the perspective back into a research conversation.</p>'+actionButton('Ask AI about '+investor+' →','ai')+'</div>'
 + '<div class="persona-grid">'+Object.keys(descriptions).map(n=>'<button type="button" class="persona-card '+(investor===n?'selected':'')+'" data-framework="'+n+'"><span class="eyebrow">'+(n==='Buffett'?'VALUE INVESTING':n==='Munger'?'MENTAL MODELS':'MARGIN OF SAFETY')+'</span><strong>'+n+'-inspired</strong><span>'+descriptions[n][0]+'</span></button>').join('')+'</div>'
 + '<div class="columns"><div class="panel"><div class="section-top"><h2>'+investor+'-inspired review</h2><span class="badge badge-demo">EDUCATIONAL SIMULATION</span></div>'
 + '<p class="page-sub">'+current[1]+'</p><div class="text-box"><b>Example research question</b><p>How would this framework assess NVIDIA, accounting for valuation and the user\'s existing portfolio?</p></div>'
 + '<button class="button button-primary" data-prompt="Apply the '+investor+' investing framework to NVIDIA and my portfolio">Continue in AI Chatbot →</button></div>'
 + '<div class="panel"><div class="section-top"><h2>Compare analytical perspectives</h2></div><div class="data-line"><b>Buffett</b><span>Quality, competitive moat, cash generation</span></div><div class="data-line"><b>Munger</b><span>Inversion, incentives, cognitive biases</span></div><div class="data-line"><b>Graham</b><span>Conservative valuation, safety margin</span></div><p class="watchlist-note">These simulations are not affiliated with, voiced by, or endorsed by real investors.</p></div></div>';
}

const pages={ai,personal,automation,community,humans};
function render(){byId('main-nav').innerHTML=NAV.map(([id,name,icon])=>`<button type="button" data-go="${id}" class="${tab===id?'active':''}"><span class="nav-icon">${icon}</span><span class="nav-text">${name}</span></button>`).join('');byId('crumb').textContent=NAV.find(x=>x[0]===tab)?.[1]||'AI Investment Chatbot';byId('page-content').innerHTML=pages[tab]();
 document.querySelectorAll('[data-go]').forEach(el=>el.addEventListener('click',()=>{tab=el.dataset.go;render();window.scrollTo(0,0)}));
 document.querySelectorAll('[data-prompt]').forEach(el=>el.addEventListener('click',()=>{tab='ai';const t=el.dataset.prompt;messages.push({user:true,message:t},{user:false,message:scriptedAnswer(t)});render();}));
 document.querySelectorAll('[data-framework]').forEach(el=>el.addEventListener('click',()=>{investor=el.dataset.framework;render()}));document.querySelectorAll('[data-video]').forEach(el=>el.addEventListener('click',openVideo));
 const stock=byId('stock-select');if(stock)stock.addEventListener('change',e=>{selected=e.target.value;render()});
 const trade=byId('trade-form');if(trade)trade.addEventListener('submit',e=>{e.preventDefault();const f=new FormData(trade),s=f.get('symbol'),n=Number(f.get('quantity')),price=TICKERS.find(t=>t.s===s).p,side=f.get('side'),fee=price*n*.0005;if(side==='buy'){if(cash<price*n+fee){byId('trade-feedback').textContent='Insufficient virtual cash.';return}cash-=price*n+fee;holdings[s]=(holdings[s]||0)+n}else{if((holdings[s]||0)<n){byId('trade-feedback').textContent='Insufficient virtual shares.';return}cash+=price*n-fee;holdings[s]-=n}render();byId('trade-feedback').textContent=`Paper trade executed: ${side} ${n} ${s} at ${usd(price)}. No real money or orders.`});
 const factorSel=byId('factor-select');if(factorSel)factorSel.addEventListener('change',e=>factor=e.target.value);
 const run=byId('quant-run');if(run)run.addEventListener('click',()=>{byId('quant-plot').innerHTML=chart(factor==='balanced'?2:factor==='momentum'?5:8);byId('quant-feedback').textContent='Illustration updated. No historical backtest or performance statistic was calculated.'});
 const form=byId('ask-form');if(form)form.addEventListener('submit',e=>{e.preventDefault();const q=new FormData(form).get('question').trim();if(!q)return;messages.push({user:true,message:q},{user:false,message:scriptedAnswer(q)});render()});
 const profileForm=byId('profile-form');if(profileForm)profileForm.addEventListener('submit',e=>{e.preventDefault();const f=new FormData(profileForm);profile={risk:f.get('risk'),horizon:f.get('horizon'),sector:f.get('sector'),experience:f.get('experience')};render();byId('profile-feedback').textContent='Demo profile updated. Your AI context will now reflect these preferences.'});
 const rule=byId('rule-form');if(rule)rule.addEventListener('submit',e=>{e.preventDefault();const f=new FormData(rule),price=Number(f.get('threshold'));if(price>0){rules.push(`${f.get('symbol')} ${f.get('direction')} ${usd(price)}`);render();byId('rule-feedback').textContent='Example rule saved in memory. No real alerts are sent.'}});
}
const video=byId('product-video'),dialog=byId('video-dialog'),error=byId('video-error');
function openVideo(){dialog.classList.add('open');dialog.setAttribute('aria-hidden','false');error.hidden=true;video.currentTime=0;const p=video.play();if(p&&p.catch)p.catch(()=>{ /* Autoplay can be blocked by browser preferences; native controls remain visible. */ });}
function closeVideo(){video.pause();dialog.classList.remove('open');dialog.setAttribute('aria-hidden','true')}
byId('open-video-top').addEventListener('click',openVideo);byId('close-video').addEventListener('click',closeVideo);dialog.addEventListener('click',e=>{if(e.target===dialog)closeVideo()});document.addEventListener('keydown',e=>{if(e.key==='Escape')closeVideo()});video.addEventListener('error',()=>{error.hidden=false});
render();
if(new URLSearchParams(location.search).has('autoplay'))window.addEventListener('load',()=>{openVideo()});