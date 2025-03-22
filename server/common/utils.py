import csv
import datetime
import time
import struct



""" Bets storage location. """
STORAGE_FILEPATH = "./bets.csv"
""" Simulated winner number in the lottery contest. """
LOTTERY_WINNER_NUMBER = 7574


""" A lottery bet registry. """
class Bet:
    def __init__(self, agency: str, first_name: str, last_name: str, document: str, birthdate: str, number: str):
        """
        agency must be passed with integer format.
        birthdate must be passed with format: 'YYYY-MM-DD'.
        number must be passed with integer format.
        """
        self.agency = int(agency)
        self.first_name = first_name
        self.last_name = last_name
        self.document = document
        self.birthdate = datetime.date.fromisoformat(birthdate)
        self.number = int(number)

""" Checks whether a bet won the prize or not. """
def has_won(bet: Bet) -> bool:
    return bet.number == LOTTERY_WINNER_NUMBER

"""
Persist the information of each bet in the STORAGE_FILEPATH file.
Not thread-safe/process-safe.
"""
def store_bets(bets: list[Bet]) -> None:
    with open(STORAGE_FILEPATH, 'a+') as file:
        writer = csv.writer(file, quoting=csv.QUOTE_MINIMAL)
        for bet in bets:
            writer.writerow([bet.agency, bet.first_name, bet.last_name,
                             bet.document, bet.birthdate, bet.number])

"""
Loads the information all the bets in the STORAGE_FILEPATH file.
Not thread-safe/process-safe.
"""
def load_bets() -> list[Bet]:
    with open(STORAGE_FILEPATH, 'r') as file:
        reader = csv.reader(file, quoting=csv.QUOTE_MINIMAL)
        for row in reader:
            yield Bet(row[0], row[1], row[2], row[3], row[4], row[5])


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
    total_lenght_bytes = read_exact(socket, 2)
    total_length = struct.unpack('>H', total_lenght_bytes)[0]
    
    data = read_exact(socket, total_length)
    
    agencia = data[0]
    offset = 1
    
    nombre_len = data[offset]
    offset += 1
    nombre = data[offset:offset + nombre_len].decode('utf-8')
    offset += nombre_len

    apellido_len = data[offset]
    offset += 1
    
    apellido = data[offset:offset + apellido_len].decode('utf-8')
    offset += apellido_len
        
    documento = data[offset:offset + 8].decode('utf-8')
    offset += 8
        
    nacimiento = data[offset:offset + 10].decode('utf-8')
    offset += 10
        
    numero = struct.unpack('>H', data[offset:offset + 2])[0]
    offset += 2
        
    return Bet(agencia, nombre, apellido, documento, nacimiento, numero)

