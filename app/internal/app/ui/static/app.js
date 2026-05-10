const roleBadge = document.getElementById("roleBadge");
const identityEmail = document.getElementById("identityEmail");
const identityFingerprint = document.getElementById("identityFingerprint");
const signatureAlgorithm = document.getElementById("signatureAlgorithm");
const encryptionAlgorithm = document.getElementById("encryptionAlgorithm");
const keyTransport = document.getElementById("keyTransport");

const sendForm = document.getElementById("sendForm");
const sendButton = document.getElementById("sendButton");
const sendStatus = document.getElementById("sendStatus");
const sendResult = document.getElementById("sendResult");

const verifyForm = document.getElementById("verifyForm");
const verifyButton = document.getElementById("verifyButton");
const verifyStatus = document.getElementById("verifyStatus");
const verifyResult = document.getElementById("verifyResult");
const downloadButton = document.getElementById("downloadButton");

const streamStatus = document.getElementById("streamStatus");
const inboundStatus = document.getElementById("inboundStatus");
const inboundSection = document.getElementById("inboundSection");
const inboundEmpty = document.getElementById("inboundEmpty");
const inboundList = document.getElementById("inboundList");
const inboundShortcut = document.getElementById("inboundShortcut");

let decryptedDocument = null;
let decryptedFileName = "decrypted-document.docx";
let decryptedMimeType =
  "application/vnd.openxmlformats-officedocument.wordprocessingml.document";
let inboundPackages = [];
let eventSource = null;
let inboundShortcutObserver = null;
let observedInboundCard = null;

boot();

async function boot() {
  await loadIdentity();
  bindForms();
  await loadInboundPackages();
  connectInboundStream();
}

async function loadIdentity() {
  try {
    const response = await fetch("/api/v1/identity");
    const payload = await response.json();

    if (!response.ok || !payload?.success) {
      throw new Error(payload?.error?.message || "Identity request failed.");
    }

    const identity = payload.data;
    const role = String(identity.role || "unknown").toLowerCase();

    roleBadge.textContent = role;
    roleBadge.className = `role-badge ${role === "recipient" ? "role-recipient" : "role-sender"}`;
    identityEmail.textContent = identity.email || "unknown";
    identityFingerprint.textContent = identity.public_key_fingerprint || "";
    signatureAlgorithm.textContent = identity.signature_algorithm || "-";
    encryptionAlgorithm.textContent = identity.encryption_algorithm || "-";
    keyTransport.textContent = identity.key_transport || "-";
  } catch (error) {
    roleBadge.textContent = "unavailable";
    roleBadge.className = "role-badge role-loading";
    identityEmail.textContent = "Identity unavailable";
    identityFingerprint.textContent = String(error.message || error);
  }
}

function bindForms() {
  sendForm.addEventListener("submit", handleSendSubmit);
  verifyForm.addEventListener("submit", handleVerifySubmit);
  downloadButton.addEventListener("click", downloadDecryptedDocument);
  inboundShortcut.addEventListener("click", jumpToLatestDecryptedDocument);
}

async function handleSendSubmit(event) {
  event.preventDefault();

  const formData = new FormData(sendForm);
  setStatus(sendStatus, "Sending document package...", "neutral");
  sendResult.textContent = "";
  sendButton.disabled = true;

  try {
    const response = await fetch("/api/v1/documents/send", {
      method: "POST",
      body: formData
    });
    const payload = await response.json();

    if (!response.ok || !payload?.success) {
      throw new Error(payload?.error?.message || "Document send failed.");
    }

    setStatus(sendStatus, "Document signed, encrypted and sent successfully.", "success");
    sendResult.textContent = JSON.stringify(payload.data, null, 2);
    sendForm.reset();
  } catch (error) {
    setStatus(sendStatus, String(error.message || error), "error");
  } finally {
    sendButton.disabled = false;
  }
}

async function handleVerifySubmit(event) {
  event.preventDefault();

  const packageInput = document.getElementById("packageFile");
  const file = packageInput.files?.[0];
  if (!file) {
    setStatus(verifyStatus, "Package JSON file is required.", "warning");
    return;
  }

  const formData = new FormData();
  formData.append("package", file);

  setStatus(verifyStatus, "Verifying and decrypting package...", "neutral");
  verifyResult.textContent = "";
  verifyButton.disabled = true;
  downloadButton.disabled = true;

  try {
    const response = await fetch("/api/v1/packages/verify-decrypt", {
      method: "POST",
      body: formData
    });
    const payload = await response.json();

    if (!response.ok || !payload?.success) {
      throw new Error(payload?.error?.message || "Package verification failed.");
    }

    const result = payload.data;
    decryptedDocument = result.decrypted_document_base64 || null;
    decryptedFileName = result?.metadata?.original_file_name || "decrypted-document.docx";
    decryptedMimeType =
      result?.metadata?.mime_type ||
      "application/vnd.openxmlformats-officedocument.wordprocessingml.document";
    downloadButton.disabled = !decryptedDocument;

    setStatus(verifyStatus, "Package verified and document decrypted successfully.", "success");
    verifyResult.textContent = JSON.stringify(result, null, 2);
  } catch (error) {
    decryptedDocument = null;
    downloadButton.disabled = true;
    setStatus(verifyStatus, String(error.message || error), "error");
  } finally {
    verifyButton.disabled = false;
  }
}

async function loadInboundPackages() {
  try {
    const response = await fetch("/api/v1/inbound/packages");
    const payload = await response.json();

    if (!response.ok || !payload?.success) {
      throw new Error(payload?.error?.message || "Inbound packages request failed.");
    }

    inboundPackages = Array.isArray(payload.data) ? payload.data : [];
    renderInboundPackages();
    if (inboundPackages.length > 0) {
      setStatus(inboundStatus, `Loaded ${inboundPackages.length} inbound package record(s).`, "success");
    } else {
      setStatus(inboundStatus, "Waiting for inbound packages.", "neutral");
    }
  } catch (error) {
    inboundPackages = [];
    renderInboundPackages();
    setStatus(inboundStatus, String(error.message || error), "error");
  }
}

function connectInboundStream() {
  if (eventSource) {
    eventSource.close();
  }

  streamStatus.textContent = "Connecting...";
  eventSource = new EventSource("/api/v1/events");

  eventSource.addEventListener("ready", () => {
    streamStatus.textContent = "Connected";
  });

  eventSource.addEventListener("inbound-package", (event) => {
    try {
      const payload = JSON.parse(event.data);
      upsertInboundPackage(payload?.package);
      const fileName = payload?.package?.original_file_name || payload?.package?.package_file_name || "package";
      setStatus(inboundStatus, `Inbound package updated: ${fileName}`, "success");
    } catch (error) {
      setStatus(inboundStatus, `Stream payload error: ${String(error.message || error)}`, "warning");
    }
  });

  eventSource.addEventListener("ping", () => {
    if (streamStatus.textContent !== "Connected") {
      streamStatus.textContent = "Connected";
    }
  });

  eventSource.onerror = () => {
    streamStatus.textContent = "Reconnecting...";
  };
}

function upsertInboundPackage(nextPackage) {
  if (!nextPackage?.mail_message_id) {
    return;
  }

  const nextID = nextPackage.mail_message_id;
  const nextItems = [];
  let replaced = false;

  for (const current of inboundPackages) {
    if (current.mail_message_id === nextID) {
      nextItems.push(nextPackage);
      replaced = true;
      continue;
    }
    nextItems.push(current);
  }

  if (!replaced) {
    nextItems.push(nextPackage);
  }

  inboundPackages = nextItems.sort(compareInboundPackages);
  renderInboundPackages();
}

function renderInboundPackages() {
  inboundList.innerHTML = "";
  const items = [...inboundPackages].sort(compareInboundPackages);

  inboundEmpty.hidden = items.length > 0;
  inboundList.hidden = items.length === 0;

  for (const item of items) {
    inboundList.appendChild(renderInboundCard(item));
  }

  syncInboundShortcut();
}

function renderInboundCard(item) {
  const card = document.createElement("article");
  card.className = "inbound-card";
  card.id = inboundCardID(item.mail_message_id);
  card.dataset.mailMessageId = item.mail_message_id || "";

  const statusClass = `status-pill status-pill-${item.status || "received"}`;
  const mailReceivedAt = formatDateTime(item.mail_received_at);
  const processedAt = formatDateTime(item.processed_at);
  const sender = escapeHTML(item.sender_email || "Unknown sender");
  const fileName = escapeHTML(item.original_file_name || item.package_file_name || "Unknown file");
  const documentID = escapeHTML(item.document_id || "Pending");
  const subject = escapeHTML(item.subject || "No subject");
  const fingerprint = escapeHTML(item.sender_public_key_fingerprint || "Pending");
  const packagePath = escapeHTML(item.package_path || "-");
  const decryptedPath = escapeHTML(item.decrypted_document_path || "-");
  const mailMessageID = encodeURIComponent(item.mail_message_id || "");
  const packageDownloadURL = `/api/v1/inbound/packages/${mailMessageID}/file?kind=package`;
  const documentDownloadURL = `/api/v1/inbound/packages/${mailMessageID}/file?kind=document`;
  const hasDocument = Boolean(item.decrypted_document_path);
  const actionsMarkup = `
    <div class="inbound-actions">
      <a class="inbound-action-link" href="${packageDownloadURL}">Скачать пакет</a>
      ${hasDocument ? `<a class="inbound-action-link inbound-action-link-primary" href="${documentDownloadURL}">Скачать документ</a>` : ""}
    </div>
  `;

  card.innerHTML = `
    <div class="inbound-header">
      <div>
        <h3 class="inbound-title">${fileName}</h3>
        <div class="faint">${subject}</div>
      </div>
      <span class="${statusClass}">${escapeHTML(item.status || "received")}</span>
    </div>

    <div class="inbound-meta">
      <div class="inbound-meta-item">
        <span class="meta-label">Sender</span>
        <strong>${sender}</strong>
      </div>
      <div class="inbound-meta-item">
        <span class="meta-label">Document ID</span>
        <strong class="mono">${documentID}</strong>
      </div>
      <div class="inbound-meta-item">
        <span class="meta-label">Received</span>
        <strong>${mailReceivedAt}</strong>
      </div>
      <div class="inbound-meta-item">
        <span class="meta-label">Processed</span>
        <strong>${processedAt}</strong>
      </div>
    </div>

    <div class="inbound-file-list">
      <span class="meta-label">Stored package</span>
      <strong class="mono">${packagePath}</strong>
      <span class="meta-label">Decrypted document</span>
      <strong class="mono">${decryptedPath}</strong>
      <span class="meta-label">Fingerprint</span>
      <strong class="mono">${fingerprint}</strong>
    </div>

    ${actionsMarkup}
  `;

  if (item.error_message) {
    const errorNode = document.createElement("div");
    errorNode.className = "inbound-error";
    errorNode.textContent = item.error_message;
    card.appendChild(errorNode);
  }

  return card;
}

function syncInboundShortcut() {
  const latestPackage = getLatestDecryptedPackage();
  if (!latestPackage) {
    hideInboundShortcut();
    observeInboundCard(null);
    return;
  }

  const targetCard = document.getElementById(inboundCardID(latestPackage.mail_message_id));
  if (!targetCard) {
    hideInboundShortcut();
    observeInboundCard(null);
    return;
  }

  observeInboundCard(targetCard);

  const title = latestPackage.original_file_name || latestPackage.package_file_name || "расшифрованный документ";
  inboundShortcut.setAttribute("title", `Перейти к документу: ${title}`);
  inboundShortcut.setAttribute("aria-label", `Перейти к документу: ${title}`);

  if (!isElementVisible(targetCard)) {
    showInboundShortcut();
  } else {
    hideInboundShortcut();
  }
}

function getLatestDecryptedPackage() {
  const items = inboundPackages
    .filter((item) => Boolean(item?.decrypted_document_path))
    .sort(compareInboundPackages);

  return items[0] || null;
}

function jumpToLatestDecryptedDocument() {
  const latestPackage = getLatestDecryptedPackage();
  if (!latestPackage) {
    return;
  }

  const targetCard = document.getElementById(inboundCardID(latestPackage.mail_message_id));
  if (!targetCard) {
    inboundSection?.scrollIntoView({ behavior: "smooth", block: "start" });
    return;
  }

  targetCard.scrollIntoView({ behavior: "smooth", block: "center" });
  pulseInboundCard(targetCard);
}

function pulseInboundCard(card) {
  if (!card) {
    return;
  }

  card.classList.remove("inbound-card-highlight");
  window.requestAnimationFrame(() => {
    card.classList.add("inbound-card-highlight");
    window.setTimeout(() => {
      card.classList.remove("inbound-card-highlight");
    }, 1300);
  });
}

function observeInboundCard(targetCard) {
  if (observedInboundCard === targetCard) {
    return;
  }

  if (inboundShortcutObserver) {
    inboundShortcutObserver.disconnect();
  }

  observedInboundCard = targetCard;
  if (!targetCard) {
    return;
  }

  inboundShortcutObserver = new IntersectionObserver(
    (entries) => {
      const [entry] = entries;
      if (!entry) {
        return;
      }

      if (entry.isIntersecting && entry.intersectionRatio >= 0.35) {
        hideInboundShortcut();
      } else if (getLatestDecryptedPackage()) {
        showInboundShortcut();
      }
    },
    {
      threshold: [0.35, 0.75]
    }
  );

  inboundShortcutObserver.observe(targetCard);
}

function isElementVisible(element) {
  if (!element) {
    return false;
  }

  const rect = element.getBoundingClientRect();
  const viewportHeight = window.innerHeight || document.documentElement.clientHeight;
  const visibleTop = Math.max(0, rect.top);
  const visibleBottom = Math.min(viewportHeight, rect.bottom);
  const visibleHeight = Math.max(0, visibleBottom - visibleTop);

  return visibleHeight >= Math.min(rect.height * 0.35, 180);
}

function showInboundShortcut() {
  inboundShortcut.hidden = false;
  inboundShortcut.classList.add("is-visible");
}

function hideInboundShortcut() {
  inboundShortcut.classList.remove("is-visible");
  inboundShortcut.hidden = true;
}

function inboundCardID(mailMessageID) {
  return `inbound-card-${String(mailMessageID || "").replaceAll(/[^a-zA-Z0-9_-]/g, "_")}`;
}

function compareInboundPackages(left, right) {
  const leftValue = Date.parse(left?.processed_at || left?.mail_received_at || 0);
  const rightValue = Date.parse(right?.processed_at || right?.mail_received_at || 0);
  return rightValue - leftValue;
}

function downloadDecryptedDocument() {
  if (!decryptedDocument) {
    return;
  }

  const binary = atob(decryptedDocument);
  const bytes = Uint8Array.from(binary, (char) => char.charCodeAt(0));
  const blob = new Blob([bytes], { type: decryptedMimeType });
  const url = URL.createObjectURL(blob);
  const link = document.createElement("a");
  link.href = url;
  link.download = decryptedFileName;
  link.click();
  URL.revokeObjectURL(url);
}

function setStatus(element, message, tone) {
  element.textContent = message;
  element.className = `status-box status-${tone}`;
}

function formatDateTime(value) {
  if (!value) {
    return "Pending";
  }

  const parsed = new Date(value);
  if (Number.isNaN(parsed.getTime())) {
    return value;
  }

  return parsed.toLocaleString("ru-RU");
}

function escapeHTML(value) {
  return String(value)
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll("\"", "&quot;")
    .replaceAll("'", "&#39;");
}
