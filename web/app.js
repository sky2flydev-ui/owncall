let ws;
let stream;
let peerId;
let room;
let peers = new Map();
let muted = false;
let cameraOff = false;

const $ = (id) => document.getElementById(id);
const rtcConfig = { iceServers: window.OWNCALL_ICE_SERVERS || [] };

$('join').onclick = join;
$('leave').onclick = leave;
$('mute').onclick = toggleMute;
$('camera').onclick = toggleCamera;

async function join() {
  room = $('room').value.trim() || 'demo-room';
  const randomId = crypto.randomUUID?.() || Math.random().toString(36).slice(2);
  const name = $('name').value.trim().replace(/[^a-zA-Z0-9_-]/g, '').slice(0, 24);
  peerId = `${name || 'guest'}-${randomId.slice(0, 8)}`;
  try {
    stream = await navigator.mediaDevices.getUserMedia({ audio: true, video: true });
    $('local').srcObject = stream;
    const protocol = location.protocol === 'https:' ? 'wss' : 'ws';
    ws = new WebSocket(`${protocol}://${location.host}/ws?room=${encodeURIComponent(room)}&peer=${encodeURIComponent(peerId)}`);
    ws.onopen = () => setStatus(`В комнате ${room}`);
    ws.onmessage = (event) => onSignal(JSON.parse(event.data));
    ws.onclose = () => setStatus('Соединение закрыто');
    ws.onerror = () => setStatus('Ошибка соединения');
    $('setup').classList.add('hidden');
    $('call').classList.remove('hidden');
  } catch (error) {
    alert(`Не удалось получить камеру/микрофон: ${error.message}`);
  }
}

async function onSignal(message) {
  if (message.type === 'peers') {
    for (const id of message.data || []) await makePeer(id, true);
  } else if (message.type === 'peer-joined') {
    setStatus('Новый участник подключился');
  } else if (message.type === 'offer') {
    const peer = await makePeer(message.from, false);
    await peer.pc.setRemoteDescription(message.data);
    await flushCandidates(peer);
    const answer = await peer.pc.createAnswer();
    await peer.pc.setLocalDescription(answer);
    send('answer', message.from, peer.pc.localDescription);
  } else if (message.type === 'answer') {
    const peer = peers.get(message.from);
    if (peer) {
      await peer.pc.setRemoteDescription(message.data);
      await flushCandidates(peer);
    }
  } else if (message.type === 'candidate') {
    const peer = peers.get(message.from);
    if (!peer) return;
    if (peer.pc.remoteDescription) await peer.pc.addIceCandidate(message.data);
    else peer.candidates.push(message.data);
  } else if (message.type === 'peer-left') {
    removePeer(message.peer);
  } else if (message.type === 'error') {
    setStatus('Ошибка: участник с таким именем уже подключён');
  }
}

async function makePeer(id, initiator) {
  if (peers.has(id)) return peers.get(id);
  const pc = new RTCPeerConnection(rtcConfig);
  const peer = { pc, id, candidates: [] };
  peers.set(id, peer);
  stream.getTracks().forEach((track) => pc.addTrack(track, stream));
  pc.onicecandidate = (event) => event.candidate && send('candidate', id, event.candidate);
  pc.ontrack = (event) => showRemote(id, event.streams[0]);
  pc.onconnectionstatechange = () => {
    if (['failed', 'closed', 'disconnected'].includes(pc.connectionState)) removePeer(id);
  };
  if (initiator) {
    const offer = await pc.createOffer();
    await pc.setLocalDescription(offer);
    send('offer', id, pc.localDescription);
  }
  return peer;
}

async function flushCandidates(peer) {
  for (const candidate of peer.candidates.splice(0)) await peer.pc.addIceCandidate(candidate);
}

function send(type, to, data) {
  if (ws?.readyState === WebSocket.OPEN) ws.send(JSON.stringify({ type, to, data }));
}

function showRemote(id, mediaStream) {
  let video = document.querySelector(`[data-peer="${CSS.escape(id)}"]`);
  if (!video) {
    const card = document.createElement('div');
    card.className = 'video-card';
    card.dataset.peer = id;
    video = document.createElement('video');
    video.autoplay = true;
    video.playsInline = true;
    card.append(video);
    const label = document.createElement('span');
    label.textContent = id.split('-')[0];
    card.append(label);
    $('remote-list').append(card);
  }
  video.srcObject = mediaStream;
}

function removePeer(id) {
  const peer = peers.get(id);
  peer?.pc.close();
  peers.delete(id);
  document.querySelector(`[data-peer="${CSS.escape(id)}"]`)?.closest('.video-card')?.remove();
}

function toggleMute() {
  muted = !muted;
  stream?.getAudioTracks().forEach((track) => { track.enabled = !muted; });
  $('mute').textContent = muted ? 'Включить микрофон' : 'Выключить микрофон';
}

function toggleCamera() {
  cameraOff = !cameraOff;
  stream?.getVideoTracks().forEach((track) => { track.enabled = !cameraOff; });
  $('camera').textContent = cameraOff ? 'Включить камеру' : 'Выключить камеру';
}

function setStatus(text) { $('status').textContent = text; }

function leave() {
  peers.forEach((peer) => peer.pc.close());
  peers.clear();
  stream?.getTracks().forEach((track) => track.stop());
  ws?.close();
  $('remote-list').replaceChildren();
  $('call').classList.add('hidden');
  $('setup').classList.remove('hidden');
}
