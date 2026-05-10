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

let decryptedDocument = null;
let decryptedFileName = "decrypted-document.docx";
let decryptedMimeType =
  "application/vnd.openxmlformats-officedocument.wordprocessingml.document";

boot();

async function boot() {
  await loadIdentity();
  bindForms();
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
