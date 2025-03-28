import socket
import logging
import signal
import time
from common.utils import *
from common.socket_utils import *
import struct
from multiprocessing import Process, Manager, Barrier, Lock, Event


BETS_MESSAGE = 1
WINNERS_REQUEST_MESSAGE = 2

class Server:
    def __init__(self, port, listen_backlog, expected_clients):
        # Initialize server socket
        self._server_socket = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        self._server_socket.bind(('', port))
        self._server_socket.listen(listen_backlog)
        self._server_socket.settimeout(1)
        self._running = True
        self._expected_clients = expected_clients
        
        # Use Manager to share state between processes
        self.manager = Manager()
        self._waiting_clients = self.manager.dict()
        self._shutdown_event = self.manager.Event()  # Event para señalizar shutdown a todos los procesos

        # Barrier to synchronize clients
        self._barrier = Barrier(expected_clients)
        self._lock = Lock()  
        self._client_processes = []

        signal.signal(signal.SIGTERM, self.shutdown)

    def run(self):
        """
        Main Server loop: Accept new connections and establish communication with a client.
        After all clients have communicated, the server sends the results to all clients.
        """

        while self._running and len(self._waiting_clients) < self._expected_clients:
            try:
                client_sock = self.__accept_new_connection()
                if client_sock:
                    logging.info("action: CREATING NEW PROCESS | result: success")
                    client_process = Process(target=self.__handle_client, args=(client_sock,))
                    client_process.start()
                    self._client_processes.append(client_process)
            except socket.timeout:
                continue
            except OSError:
                break
            
            # Verifica si hay señal de shutdown para salir del bucle
            if self._shutdown_event.is_set():
                break

        if not self._shutdown_event.is_set():
            logging.info("action: sorteo | result: success")
            
            for process in self._client_processes:
                process.join()

            try:
                self.__send_winners()
            except OSError as e:
                logging.error(f"action: run_server | result: fail | error: {e}")
        else:
            logging.info("action: run_interrupted | result: success")

    def __handle_client(self, client_sock):
        keep_open = False
        try:
            while self._running:
                msg = read_exact(client_sock, 1)
                if not msg:
                    break

                message_type = msg[0]
                if message_type == BETS_MESSAGE:
                    self.__handle_bets_message(client_sock)
                elif message_type == WINNERS_REQUEST_MESSAGE:
                    agency = self.__handle_winners_request_message(client_sock)
                    keep_open = True
                    
                    if self._shutdown_event.is_set():
                        break
                    
                    try:
                        # Durante un shutdown, usamos un timeout pequeño para liberar los procesos
                        timeout = 0.1 if self._shutdown_event.is_set() else None
                        self._barrier.wait(timeout=timeout)
                    except Exception as e:
                        if not self._shutdown_event.is_set():
                            logging.error(f"action: barrier_wait | result: fail | error: {e}")
                    break
                else:
                    logging.warning("action: unknown_message_type")
                    break
        except Exception as e:
            if not self._shutdown_event.is_set():
                logging.error(f"action: handle_client | result: fail | error: {e}")
        finally:
            if not keep_open or self._shutdown_event.is_set():
                client_sock.close()

    def __handle_bets_message(self, client_sock):
        try:
            bets = deserialize_bet_batch(client_sock)
            bets_quantity = len(bets)
            addr = client_sock.getpeername()
            logging.info(f'action: receive_message | result: success | ip: {addr[0]}')

            self.__store_bets_secure(bets, self._lock)

            logging.info(f'action: apuesta_recibida | result: success | cantidad: {bets_quantity}')
            confirmation = struct.pack('>B', 1)
            write_exact(client_sock, confirmation)
        except OSError as e:
            logging.error(f"action: receive_message | result: fail | error: {e}")

    def __store_bets_secure(self, bets, lock):
        with lock:
            store_bets(bets)

    def __handle_winners_request_message(self, client_sock):
        try:
            agency = read_exact(client_sock, 1)[0]
            self._waiting_clients[agency] = client_sock
            return agency
        except OSError as e:
            raise e

    def __accept_new_connection(self):
        logging.info('action: accept_connections | result: in_progress')
        c, addr = self._server_socket.accept()
        logging.info(f'action: accept_connections | result: success | ip: {addr[0]}')
        return c

    def __send_winners(self):
        winners = {}
        for i in range(1, self._expected_clients + 1):
            winners[i] = []
        for bet in load_bets():
            if has_won(bet):
                winners[bet.agency].append(bet)
        for agency, socket in self._waiting_clients.items():
            if not self._shutdown_event.is_set():
                self.__send_winners_to_agency(socket, winners[agency])

    def __send_winners_to_agency(self, agency_socket, bets):
        try:
            length = len(bets)
            agency_socket.send(struct.pack('>H', length))

            for bet in bets:
                document = bet.document.encode('utf8')
                write_exact(agency_socket, document)
        except Exception as e:
            logging.error(f"action: send_winners | result: fail | error: {e}")
        finally:
            agency_socket.close()

    def shutdown(self, signum, frame):
        logging.info("action: shutdown | result: in_progress")
        
        self._shutdown_event.set()
        self._running = False
        
        start_time = time.time()
        shutdown_timeout = 1.0
        
        try:
            self._barrier.abort()
            logging.info("action: barrier_abort | result: success")
        except Exception as e:
            logging.error(f"action: barrier_abort | result: fail | error: {e}")
        
        for process in self._client_processes:
            remaining_time = max(0, shutdown_timeout - (time.time() - start_time))
            process.join(timeout=remaining_time)
            if process.is_alive() and time.time() - start_time >= shutdown_timeout:
                process.terminate()
        
        try:
            self._server_socket.close()
        except Exception as e:
            logging.error(f"action: close_server_socket | result: fail | error: {e}")

        logging.info('action: exit | result: success')