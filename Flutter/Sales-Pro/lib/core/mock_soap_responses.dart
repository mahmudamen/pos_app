class MockSoapResponses {
  static String getResponse(String methodName) {
    switch (methodName) {
      case 'Select_User':
        return _selectUserResponse;
      case 'Get_customers':
        return _customersResponse;
      case 'Get_Price':
        return _pricesResponse;
      case 'Get_Store':
        return _storesResponse;
      case 'Update_User_Store':
        return _updateUserStoreResponse;
      case 'Get_Balance':
        return _balanceResponse;
      default:
        return _defaultResponse;
    }
  }

  static const _selectUserResponse = '''
<?xml version="1.0" encoding="utf-8"?>
<soap:Envelope xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xmlns:xsd="http://www.w3.org/2001/XMLSchema" xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
  <soap:Body>
    <Select_UserResponse xmlns="http://tempuri.org/">
      <Select_UserResult>
        <IDUser_>1</IDUser_>
        <SaleMan_>1</SaleMan_>
        <IDTStore_>1</IDTStore_>
        <Name_Store_>Main Store</Name_Store_>
      </Select_UserResult>
    </Select_UserResponse>
  </soap:Body>
</soap:Envelope>
''';

  static const _customersResponse = '''
<?xml version="1.0" encoding="utf-8"?>
<soap:Envelope xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xmlns:xsd="http://www.w3.org/2001/XMLSchema" xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
  <soap:Body>
    <Get_customersResponse xmlns="http://tempuri.org/">
      <Get_customersResult>
        <diffgr:diffgram xmlns:msdata="urn:schemas-microsoft-com:xml-msdata" xmlns:diffgr="urn:schemas-microsoft-com:xml-diffgram-v1">
          <NewDataSet xmlns="">
            <TCustomer diffgr:id="TCustomer1" msdata:rowOrder="0">
              <ID_>101</ID_>
              <Barcode_>C001</Barcode_>
              <Name_>Ahmed Salem</Name_>
              <credit_>1500.50</credit_>
            </TCustomer>
            <TCustomer diffgr:id="TCustomer2" msdata:rowOrder="1">
              <ID_>102</ID_>
              <Barcode_>C002</Barcode_>
              <Name_>Fatima Ali</Name_>
              <credit_>2300.00</credit_>
            </TCustomer>
            <TCustomer diffgr:id="TCustomer3" msdata:rowOrder="2">
              <ID_>103</ID_>
              <Barcode_>C003</Barcode_>
              <Name_>Mohammed Hassan</Name_>
              <credit_>750.25</credit_>
            </TCustomer>
          </NewDataSet>
        </diffgr:diffgram>
      </Get_customersResult>
    </Get_customersResponse>
  </soap:Body>
</soap:Envelope>
''';

  static const _pricesResponse = '''
<?xml version="1.0" encoding="utf-8"?>
<soap:Envelope xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xmlns:xsd="http://www.w3.org/2001/XMLSchema" xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
  <soap:Body>
    <Get_PriceResponse xmlns="http://tempuri.org/">
      <Get_PriceResult>
        <diffgr:diffgram xmlns:msdata="urn:schemas-microsoft-com:xml-msdata" xmlns:diffgr="urn:schemas-microsoft-com:xml-diffgram-v1">
          <NewDataSet xmlns="">
            <TablePrice diffgr:id="TablePrice1" msdata:rowOrder="0">
              <ID_>P001</ID_>
              <Barcode_>123456789</Barcode_>
              <Name_>Laptop HP 15</Name_>
              <Price_>2500.00</Price_>
              <QU_>15</QU_>
            </TablePrice>
            <TablePrice diffgr:id="TablePrice2" msdata:rowOrder="1">
              <ID_>P002</ID_>
              <Barcode_>987654321</Barcode_>
              <Name_>Samsung Galaxy S23</Name_>
              <Price_>3200.50</Price_>
              <QU_>8</QU_>
            </TablePrice>
            <TablePrice diffgr:id="TablePrice3" msdata:rowOrder="2">
              <ID_>P003</ID_>
              <Barcode_>456789123</Barcode_>
              <Name_>Apple AirPods Pro</Name_>
              <Price_>899.99</Price_>
              <QU_>25</QU_>
            </TablePrice>
            <TablePrice diffgr:id="TablePrice4" msdata:rowOrder="3">
              <ID_>P004</ID_>
              <Barcode_>789123456</Barcode_>
              <Name_>Dell Monitor 27"</Name_>
              <Price_>1200.00</Price_>
              <QU_>12</QU_>
            </TablePrice>
            <TablePrice diffgr:id="TablePrice5" msdata:rowOrder="4">
              <ID_>P005</ID_>
              <Barcode_>321654987</Barcode_>
              <Name_>Logitech Wireless Mouse</Name_>
              <Price_>45.75</Price_>
              <QU_>50</QU_>
            </TablePrice>
          </NewDataSet>
        </diffgr:diffgram>
      </Get_PriceResult>
    </Get_PriceResponse>
  </soap:Body>
</soap:Envelope>
''';

  static const _storesResponse = '''
<?xml version="1.0" encoding="utf-8"?>
<soap:Envelope xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xmlns:xsd="http://www.w3.org/2001/XMLSchema" xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
  <soap:Body>
    <Get_StoreResponse xmlns="http://tempuri.org/">
      <Get_StoreResult>
        <diffgr:diffgram xmlns:msdata="urn:schemas-microsoft-com:xml-msdata" xmlns:diffgr="urn:schemas-microsoft-com:xml-diffgram-v1">
          <NewDataSet xmlns="">
            <TableStore diffgr:id="TableStore1" msdata:rowOrder="0">
              <ID_>1</ID_>
              <Name_>Main Store</Name_>
            </TableStore>
            <TableStore diffgr:id="TableStore2" msdata:rowOrder="1">
              <ID_>2</ID_>
              <Name_>Branch Store</Name_>
            </TableStore>
          </NewDataSet>
        </diffgr:diffgram>
      </Get_StoreResult>
    </Get_StoreResponse>
  </soap:Body>
</soap:Envelope>
''';

  static const _updateUserStoreResponse = '''
<?xml version="1.0" encoding="utf-8"?>
<soap:Envelope xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xmlns:xsd="http://www.w3.org/2001/XMLSchema" xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
  <soap:Body>
    <Update_User_StoreResponse xmlns="http://tempuri.org/">
      <Update_User_StoreResult>true</Update_User_StoreResult>
    </Update_User_StoreResponse>
  </soap:Body>
</soap:Envelope>
''';

  static const _balanceResponse = '''
<?xml version="1.0" encoding="utf-8"?>
<soap:Envelope xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xmlns:xsd="http://www.w3.org/2001/XMLSchema" xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
  <soap:Body>
    <Get_BalanceResponse xmlns="http://tempuri.org/">
      <Get_BalanceResult>15750.50</Get_BalanceResult>
    </Get_BalanceResponse>
  </soap:Body>
</soap:Envelope>
''';

  static const _defaultResponse = '''
<?xml version="1.0" encoding="utf-8"?>
<soap:Envelope xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xmlns:xsd="http://www.w3.org/2001/XMLSchema" xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
  <soap:Body>
    <Response>
      <Result>true</Result>
    </Response>
  </soap:Body>
</soap:Envelope>
''';
}
