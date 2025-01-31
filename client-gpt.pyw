import socket
import json
import threading
import tkinter as tk
from tkinter import scrolledtext, filedialog
from datetime import datetime

# Connect to the chat server
def connect_to_server(host='127.0.0.1', port=8888, name=""):
    global client_socket
    client_socket = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    client_socket.connect((host, port))
    client_socket.send(name.encode())
    return client_socket

def send_file(file_path):
    with open(file_path, "rb") as file:
        file_data = file.read()
    header = {"type": "file", "filename": file_path.split("/")[-1], "length": len(file_data)}
    client_socket.send(json.dumps(header).encode())
    client_socket.sendall(file_data)

def choose_file():
    file_path = filedialog.askopenfilename()
    if file_path:
        send_file(file_path)

def receive_file(client_socket, data_length):
    data = b''
    while len(data) < data_length:
        packet = client_socket.recv(1024)
        if not packet:
            break
        data += packet
    return data

def save_file(client_name, file_data):
    file_path = f'received_files/{client_name}_{datetime.now().strftime("%Y%m%d%H%M%S")}'
    with open(file_path, 'wb') as file:
        file.write(file_data)
    print(f"File received and saved to {file_path}")

def set_name():
    name = name_entry.get()
    name_entry.config(state="disabled")
    name_button.pack_forget()
    client_socket.send(name.encode())
    timestamp = datetime.now().strftime("%H:%M:%S")
    message_display.insert("end", f"[{timestamp}] Welcome to the chat {name}\n", "system")
    message_display.see("end")
    window.update()

def receive_messages():
    print('here')
    while True:
        data = client_socket.recv(1024)
        print(data)
        if not data:
            break
        message = json.loads(data.decode())
        message_type = message["type"]
        #message_length = message["length"]

        if message_type == "text":
            window.after(0, update_message_display, f"[{message['timestamp']}] {message['name']}: {message['text']}\n")
        elif message_type == "file":
            file_data = receive_file(client_socket, message_length)
            save_file(client_name, file_data)

def update_message_display(msg):
    message_display.insert("end", msg)
    message_display.see("end")
    window.update()

def send_message():
    message_text = message_entry.get()
    if message_text:
        message = {"text": message_text, "type": "text"}
        client_socket.send(json.dumps(message).encode())
        timestamp = datetime.now().strftime("%H:%M:%S")
        window.after(0, update_message_display, f"[{timestamp}] You: {message['text']}\n")
        message_entry.delete(0, "end")
        window.update()

def create_windows():
    global window, name_entry, name_button, message_display, message_entry
    window = tk.Tk()
    window.title("Chat Client")

    name_frame = tk.Frame(window)
    name_frame.pack(fill="x")
    name_label = tk.Label(name_frame, text="Enter your name:")
    name_label.pack(side="left")
    name_entry = tk.Entry(name_frame)
    name_entry.pack(side="left")
    name_button = tk.Button(name_frame, text="Set Name", command=set_name)
    name_button.pack(side="left")
    
    send_file_button = tk.Button(window, text="Send File", command=choose_file)
    send_file_button.pack()
    
    message_display = scrolledtext.ScrolledText(window, wrap=tk.WORD)
    message_display.tag_config('sender', foreground="#228B22")
    message_display.tag_config('system', foreground="#FF5733")
    message_display.pack(fill="both", expand=True)

    message_entry = tk.Entry(window)
    message_entry.pack(fill="x")

    send_button = tk.Button(window, text="Send", command=send_message)
    send_button.pack()

def start_daemon_thread():
    global message_thread
    message_thread = threading.Thread(target=receive_messages)
    message_thread.daemon = True
    message_thread.start()

def handle_cleanup():
    try:
        client_socket.shutdown(socket.SHUT_RDWR)
        client_socket.close()
    except Exception as e:
        print(f"Error during cleanup: {e}")

def main():
    try:
        connect_to_server()
        create_windows()
        start_daemon_thread()
        while True:
            window.mainloop()
    finally:
        handle_cleanup()

if __name__ == '__main__':
    main()
