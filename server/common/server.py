import socket
import logging
import signal
from common.utils import *
from common.socket_utils import *
import struct
from multiprocessing import Process, Manager, Barrier


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
        self._waiting_clients = {}

        # Use Manager to share state between processes
        self.manager = Manager()
        self._waiting_clients = self.manager.dict()

        # Barrier to synchronize clients
        self._barrier = Barrier(expected_clients)

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
            except socket.timeout:
                continue
            except OSError:
                break

        logging.info("action: sorteo | result: success")
        
        try:
            self.__send_winners()
        except OSError as e:
            logging.error(f"action: run_server | result: fail | error: {e}")

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
                    self.__handle_winners_request_message(client_sock)
                    keep_open = True
                    self._barrier.wait()
                    break
                else:
                    logging.warning("action: unknown_message_type")
                    break
        except Exception as e:
            logging.error(f"action: handle_client | result: fail | error: {e}")
        finally:
            if not keep_open:
                client_sock.close()

    def __get_message_type(self, client_sock):
        return read_exact(client_sock, 1)[0]

    def __handle_bets_message(self, client_sock):
        """
        Read message from a specific client socket and closes the socket

        If a problem arises in the communication with the client, the
        client socket will also be closed
        """
        try:
            bets = deserialize_bet_batch(client_sock)
            bets_quantity = len(bets)
            addr = client_sock.getpeername()
            logging.info(f'action: receive_message | result: success | ip: {addr[0]}')
            store_bets(bets)
            logging.info(f'action: apuesta_recibida | result: success | cantidad: {bets_quantity}')
            confirmation = struct.pack('>B', 1)
            write_exact(client_sock, confirmation)
        except OSError as e:
            logging.error(f"action: receive_message | result: fail | error: {e}")

    def __handle_winners_request_message(self, client_sock):
        """
        Handle the winners request and add the client to the waiting list
        """
        try:
            agency = read_exact(client_sock, 1)[0]
            self._waiting_clients[agency] = client_sock
            return agency
        except OSError as e:
            raise e

    def __accept_new_connection(self):
        """
        Accept new connections

        This function blocks until a connection to a client is made.
        When a connection is made, it returns the created connection.
        """
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
            self.__send_winners_to_agency(socket, winners[agency])

    def __send_winners_to_agency(self, agency_socket, bets):
        length = len(bets)
        agency_socket.send(struct.pack('>H', length))

        for bet in bets:
            document = bet.document.encode('utf8')
            write_exact(agency_socket, document)

        agency_socket.close()

    def shutdown(self, signum, frame):
        self._running = False
        self._server_socket.close()
        logging.info('action: exit | result: success')
