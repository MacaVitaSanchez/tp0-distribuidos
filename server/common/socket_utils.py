import struct
from common.utils import *

BETS_QUANTITY_LENGTH = 1
LEN_TOTAL_LENGTH = 2
AGENCY_LENGTH = 1
DNI_LENGTH = 8
BIRTH_LENGTH = 10
NUMBER_LENGTH = 2

"""
Function to write to a socket and ensure the written amount is as expected.
Made to avoid short writes.
"""
def write_exact(socket, data):
    sent_bytes = 0
    while sent_bytes < len(data):
        sent_bytes += socket.send(data[sent_bytes:])

"""
Function to read from a socket and ensure the read amount is as expected.
Made to avoid short reads.
"""
def read_exact(socket, length):
    data = bytearray()
    while len(data) < length:
        packet = socket.recv(length - len(data))
        if not packet:
            return None
        data.extend(packet)
    return data

"""
Function to deserialize a bet from a socket connection.
Returns the deserialized bet in the form of a Bet object.
"""
def deserialize_bet(socket):
    total_lenght_bytes = read_exact(socket, LEN_TOTAL_LENGTH)
    total_length = struct.unpack('>H', total_lenght_bytes)[0]
    
    data = read_exact(socket, total_length)
    
    agencia = data[0]
    offset = AGENCY_LENGTH
    
    nombre_len = data[offset]
    offset += 1
    nombre = data[offset:offset + nombre_len].decode('utf-8')
    offset += nombre_len

    apellido_len = data[offset]
    offset += 1
    
    apellido = data[offset:offset + apellido_len].decode('utf-8')
    offset += apellido_len
        
    documento = data[offset:offset + DNI_LENGTH].decode('utf-8')
    offset += DNI_LENGTH
        
    nacimiento = data[offset:offset + BIRTH_LENGTH].decode('utf-8')
    offset += BIRTH_LENGTH
        
    numero = struct.unpack('>H', data[offset:offset + NUMBER_LENGTH])[0]
        
    return Bet(agencia, nombre, apellido, documento, nacimiento, numero)


def deserialize_bet_batch(socket):
    cantidad = read_exact(socket, BETS_QUANTITY_LENGTH)[0]
    bets = []
    for i in range(cantidad):
        bet = deserialize_bet(socket)
        bets.append(bet)
    return bets