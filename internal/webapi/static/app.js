document.addEventListener('alpine:init', () => {


  Alpine.data('chat', () => ({
    ws: null,
    API_URL: '',
    messages: [],
    events: [],
    files: [],
    status: 'Disconnected',
    newMessage: '',
    showModal: false,
    modalMessage: '',
    currentConfirmation: null,

    init() {
      // TODO fix modal issue
      this.initWebSocket();
      this.refreshFileList();

      // Configure marked options
      marked.setOptions({
        highlight: function (code, lang) {
          if (Prism.languages[lang]) {
            return Prism.highlight(code, Prism.languages[lang], lang);
          }
          return code;
        }
      });
    },
    // async loadModal() {
    //   try {
    //     const response = await fetch('/modal-component.html');
    //     const text = await response.text();
    //     const tempDiv = document.createElement('div');
    //     tempDiv.innerHTML = text;
    //
    //     const modalContent = tempDiv.querySelector('#modal-template').content;
    //     document.body.appendChild(modalContent);
    //     this.modalLoaded = true;
    //   } catch (error) {
    //     console.error('Error loading modal:', error);
    //   }
    // },
    initWebSocket() {
      this.ws = new WebSocket('ws://localhost:8080/ws');

      this.ws.onopen = () => {
        this.logEvent('WebSocket connected');
        this.status = 'Connected';
      };

      this.ws.onclose = () => {
        this.logEvent('WebSocket disconnected');
        this.status = 'Disconnected';
        setTimeout(() => this.initWebSocket(), 5000);
      };

      this.ws.onmessage = (event) => {
        const data = JSON.parse(event.data);
        this.handleEvent(data);
      };
    },

    handleEvent(event) {
      console.log('Event:', event);
      this.logEvent(`Event received: ${event.Type}`);

      switch (event.Type) {
        case 0:
          this.addMessage('user', event.Payload.Content);
          break;
        case 1:
          this.logEvent(`Error: ${JSON.stringify(event.Payload)}`, 'error');
          break;
        case 2:
          this.addMessage('assistant', event.Payload);
          break;
        case 3:
          this.status = event.Payload.New;
          break;
        case 4:
          this.refreshFileList();
          break;
        case "confirmation_request":
          this.handleConfirmationRequest(event.Payload);
          break;
      }
    },

    renderMessage(content) {
      try {
        const rendered = marked.parse(content);
        // Use setTimeout to trigger Prism highlighting after the content is rendered
        setTimeout(() => {
          Prism.highlightAll();
        }, 0);
        return rendered;
      } catch (error) {
        console.error('Rendering error:', error);
        return content;
      }
    },

    async sendMessage() {
      if (!this.newMessage.trim()) return;

      await fetch(`${this.API_URL}/api/chat`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ message: this.newMessage }),
      });

      this.newMessage = '';
    },

    addMessage(type, content) {


      if (type === 'assistant' && this.messages.length > 0) {
        const lastMessage = this.messages[this.messages.length - 1];
        if (lastMessage.type === 'assistant') {
          // Concatenate with the previous message
          lastMessage.content += content.toString();
          this.$nextTick(() => {
            const chatHistory = document.getElementById('chat-history');
            chatHistory.scrollTop = chatHistory.scrollHeight;
            Prism.highlightAll();
          });
          return;
        }
      }


      this.messages.push({
        id: Date.now(),
        type,
        content
      });

      this.$nextTick(() => {
        const chatHistory = document.getElementById('chat-history');
        chatHistory.scrollTop = chatHistory.scrollHeight;
        Prism.highlightAll(); // Highlight any code blocks in the new message
      });

    },

    async uploadFiles() {
      const fileInput = document.getElementById('file-input');
      const formData = new FormData();

      for (const file of fileInput.files) {
        formData.append('files', file);
      }

      try {
        await fetch(`${this.API_URL}/api/files`, {
          method: 'POST',
          body: formData,
        });
        fileInput.value = '';
        this.refreshFileList();
      } catch (error) {
        this.logEvent(`Error uploading files: ${error}`, 'error');
      }
    },

    async refreshFileList() {
      try {
        const response = await fetch(`${this.API_URL}/api/files`);
        const files = await response.json();
        this.files = Object.entries(files).map(([name, info]) => ({ name, ...info }));
      } catch (error) {
        this.logEvent(`Error fetching files: ${error}`, 'error');
      }
    },

    async removeFile(filename) {
      try {
        await fetch(`${this.API_URL}/api/files/${filename}`, {
          method: 'DELETE',
        });
        this.refreshFileList();
      } catch (error) {
        this.logEvent(`Error removing file: ${error}`, 'error');
      }
    },

    logEvent(message, type = 'info') {
      this.events.push({
        id: Date.now(),
        message: `[${new Date().toISOString()}] ${message}`,
        type
      });
    },

    handleConfirmationRequest(confirmation) {
      console.log('Confirmation request:', confirmation);
      console.log('Current showModal value:', this.showModal);
      this.currentConfirmation = confirmation;
      this.modalMessage = confirmation.Message;
      // Use Alpine's $nextTick to ensure DOM updates
      this.$nextTick(() => {
        this.showModal = true;
        console.log('Modal should be visible now');
      });
      console.log('New showModal value:', this.showModal);
    },

    async handleModalResponse(approved) {
      if (!this.currentConfirmation) return;

      const response = {
        type: "user_response",
        payload: {
          id: this.currentConfirmation.ID,
          approved
        }
      };

      // User response over WS 
      // this.ws.send(JSON.stringify(response));


      // User response over HTTP
      try {
        await fetch(`${this.API_URL}/api/confirmation`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(response.payload),
        });
      } catch (error) {
        cpnsole.error('Error sending confirmation response:', error);
        this.logEvent(`Error sending confirmation response: ${error}`, 'error');
      }

      this.logEvent(`Confirmation ${this.currentConfirmation.ID} response: ${approved ? 'Approved' : 'Rejected'}`);
      this.showModal = false;
      this.currentConfirmation = null;
    }
  }));
});


