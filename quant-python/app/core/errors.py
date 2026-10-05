class QuantServiceError(Exception):
    def __init__(self, message: str, code: int = 500, data=None):
        super().__init__(message)
        self.message = message
        self.code = code
        self.data = data


class DataSourceError(QuantServiceError):
    pass


class ValidationError(QuantServiceError):
    pass
