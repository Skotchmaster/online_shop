from sqlalchemy import Column, Float, Integer, CheckConstraint, String
from connect_database import Base

class Product(Base):
    __tablename__ = "Product"

    ID = Column(Integer, primary_key=True, index=True)
    Name = Column(String, unique=True, nullable=False, index=True)
    Description = Column(String, unique=True, nullable=False)
    Price = Column(Float, nullable=False)
    Count = Column(Integer)

    __table_args__ = (
        CheckConstraint("price >= 0", name="chk_price_non_negative"),
        CheckConstraint("count >= 0", name="chk_count_non_negative"),
    )

    def __repr__(self):
        return f"<Product(name={self.name!r}, description={self.Description!r}, price={self.Price}, count={self.Count})>"
    
class Order(Base):

    __tablename__ = "Orders"

    ID = Column(Integer, primary_key=True, index=True)
    Name = Column(String, nullable=False, index=True)
    Address = Column(String, nullable=False)
    Phone_Number = Column(Float, nullable=False)

    def __repr__(self):
        return f"<Order(name={self.name!r}, address={self.Address!r}, phone number={self.Phone_Number})>"
