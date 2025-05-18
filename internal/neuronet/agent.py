from typing import Dict

from langchain.tools import tool
from langchain_openai import ChatOpenAI
from langgraph.prebuilt import create_react_agent
from langgraph.checkpoint.memory import MemorySaver

from internal.neuronet.database.connect_database import SessionLocal
from internal.neuronet.database.models_db import Order, Product


system_promt = "ты бот продавец-консультант, ты вежлив. Твоя задача продать пользователю телефон, получив от него заказ. Если тебе не хватает каких-то данных," \
" запрашивай их у пользователя. ОБЯЗАТЕЛЬНО ПРИ ОФОРМЛЕНИИ ЗАКАЗА УТОЧНИ ИМЯ ПОКУПАТЕЛЯ, ЕГО АДРЕС И НОМЕР ТЕЛЕФОНА."

@tool
def get_all_products():
    '''
    возвращает названия всех продуктов, доступных к покупке
    '''
    with SessionLocal() as db:
        rows = db.query(Product.name).all()
    return ", ".join(name for (name,) in rows)

@tool
def get_product(phone_name: str) -> Dict:
    '''
    выводит всю информацию о продукте

    Args:
        phone_name (str): точное название модели телефона

    Returns:
        Dict: словарь с информацией о продукте (цена, характерисики, описание)    
    '''
    with SessionLocal() as db:
        product = (
            db.query(Product)
              .filter(Product.name == phone_name.strip())
              .first()
        )
    if not product:
        return {}
    return {
        "id":          Product.ID,
        "name":        Product.Name,
        "description": Product.Description,
        "price":       Product.Price,
        "count":       Product.Count,
    }

@tool
def find_product_by_price(min_price: float, max_price: float) -> str:
    with SessionLocal as db:
         rows = (
            db.query(Product.name)
              .filter(Product.price.between(min_price, max_price))
              .all()
        )
    return ", ".join(name for (name,) in rows) or "В этом диапазоне ничего нет"

@tool
def find_product_by_feature(feature: str) -> str:
    """
    Ищет в описании телефонов слово feature и возвращает подходящие модели.
    """
    with SessionLocal() as db:
        rows = (
            db.query(Product.name)
              .filter(Product.description.ilike(f"%{feature}%"))
              .all()
        )
    return ", ".join(name for (name,) in rows) or "Ничего не найдено"



@tool
def create_order(name: str, phone_number: str, address: str) -> None:
    '''
    создает заказ на покупку продукта
    '''
    with SessionLocal as db:
        order = Order(
            Name = name,
            Phone_Number = phone_number,
            Address = address,
        )
    db.add(order)
    db.commit()
    db.refresh(order)    

tools = [get_all_products, get_product, create_order, find_product_by_price, find_product_by_feature]    

llm = ChatOpenAI(
        model="deepseek-r1-distill-qwen-7b",
        base_url="http://127.0.0.1:1234/v1",
        api_key="not-needed",
        temperature=0.2, 
    )

agent = create_react_agent(
    model = llm,
    tools = tools,
    checkpointer = MemorySaver(), 
    state_modifier = system_promt,
)

def chat(thread_id: str):
    config = {"configurable": {"thread_id": thread_id}}
    while True:
        rq = input("\n human: ")
        if rq == "":
            break
        resp = agent.invoke({"messages": [("user", rq)]}, config=config)
        print("AI: ", resp["messages"][-1].content)

chat("1111")       