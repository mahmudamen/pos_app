import 'dart:convert';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';

/// App localization class for managing translations
class AppLocalizations {
  final Locale locale;
  Map<String, String> _localizedStrings = {};

  AppLocalizations(this.locale);

  /// Helper method to keep the code in the widgets concise
  static AppLocalizations of(BuildContext context) {
    return Localizations.of<AppLocalizations>(context, AppLocalizations)!;
  }

  /// Static member to have a simple access to the delegate from the MaterialApp
  static const LocalizationsDelegate<AppLocalizations> delegate =
      _AppLocalizationsDelegate();

  /// Load the language JSON file from the "assets/lang" folder
  Future<bool> load() async {
    try {
      // Load the language JSON file from the "assets/lang" folder
      String jsonString = await rootBundle.loadString(
        'assets/lang/${locale.languageCode}.json',
      );
      Map<String, dynamic> jsonMap = json.decode(jsonString);

      _localizedStrings = jsonMap.map((key, value) {
        return MapEntry(key, value.toString());
      });

      return true;
    } catch (e) {
      debugPrint('Error loading localizations: $e');
      return false;
    }
  }

  /// This method will be called from every widget which needs a localized text
  String translate(String key, {Map<String, String>? params}) {
    String translation = _localizedStrings[key] ?? key;

    // Replace parameters if provided
    if (params != null) {
      params.forEach((paramKey, paramValue) {
        translation = translation.replaceAll('{$paramKey}', paramValue);
      });
    }

    return translation;
  }

  // Convenience getters for common translations
  String get appName => translate('appName');
  String get appFullName => translate('appFullName');
  String get appVersion => translate('appVersion');
  String get salesProVersion => translate('salesProVersion');

  String get welcome => translate('welcome');
  String get welcomeBack => translate('welcomeBack');
  String get login => translate('login');
  String get signup => translate('signup');
  String get logout => translate('logout');
  String get logoutConfirm => translate('logoutConfirm');
  String get logoutFailed => translate('logoutFailed');
  String get rememberMe => translate('rememberMe');
  String get enterUsername => translate('enterUsername');
  String get enterPassword => translate('enterPassword');
  String get invalidCredentials => translate('invalidCredentials');
  String get passwordRecoverySoon => translate('passwordRecoverySoon');

  String get newInvoice => translate('newInvoice');
  String get salesInvoice => translate('salesInvoice');
  String get savedDrafts => translate('savedDrafts');
  String get noSavedDrafts => translate('noSavedDrafts');
  String get startNewInvoice => translate('startNewInvoice');
  String get draftDeleted => translate('draftDeleted');
  String get draftUpdated => translate('draftUpdated');
  String get invoiceSavedAsDraft => translate('invoiceSavedAsDraft');
  String get invoiceSentSuccessfully => translate('invoiceSentSuccessfully');
  String get sendInvoice => translate('sendInvoice');
  String get saveAsDraft => translate('saveAsDraft');
  String get sending => translate('sending');

  String get grandTotal => translate('grandTotal');
  String grandTotalValue(String amount, String currency) => translate(
    'grandTotalValue',
    params: {'amount': amount, 'currency': currency},
  );
  String get receivedAmount => translate('receivedAmount');
  String get customerBalance => translate('customerBalance');
  String customerCredit(String credit) =>
      translate('customerCredit', params: {'credit': credit});
  String get noCustomer => translate('noCustomer');
  String get selectCustomer => translate('selectCustomer');
  String get selectStore => translate('selectStore');
  String get selectDefaultWarehouse => translate('selectDefaultWarehouse');
  String get noDefaultWarehouse => translate('noDefaultWarehouse');
  String get defaultWarehouse => translate('defaultWarehouse');
  String warehouseSaved(String storeName) =>
      translate('warehouseSaved', params: {'storeName': storeName});
  String get warehouseSaveFailed => translate('warehouseSaveFailed');

  String get searchItems => translate('searchItems');
  String get searchByNameBarcodeId => translate('searchByNameBarcodeId');
  String get searchByNameBarcodeCode => translate('searchByNameBarcodeCode');
  String get scanBarcode => translate('scanBarcode');
  String get barcodeScanner => translate('barcodeScanner');
  String get noResultsFound => translate('noResultsFound');
  String get noMatchingItems => translate('noMatchingItems');
  String get noItemsFound => translate('noItemsFound');
  String get itemNotFound => translate('itemNotFound');
  String itemId(String id) => translate('itemId', params: {'id': id});
  String itemWithBarcode(String name, String barcode) =>
      translate('itemWithBarcode', params: {'name': name, 'barcode': barcode});
  String itemBarcode(String barcode) =>
      translate('itemBarcode', params: {'barcode': barcode});
  String itemStock(String stock) =>
      translate('itemStock', params: {'stock': stock});

  String get quantityMustBeGreaterThanZero =>
      translate('quantityMustBeGreaterThanZero');
  String get quantityAtLeastOne => translate('quantityAtLeastOne');
  String quantityExceedsStock(String stock) =>
      translate('quantityExceedsStock', params: {'stock': stock});
  String get selectQuantity => translate('selectQuantity');
  String get availableStock => translate('availableStock');

  String get addAtLeastOneItem => translate('addAtLeastOneItem');
  String get addAtLeastOneItemDraft => translate('addAtLeastOneItemDraft');
  String get addItemsHint => translate('addItemsHint');
  String get invoiceEmpty => translate('invoiceEmpty');

  String get note => translate('note');
  String get addNote => translate('addNote');
  String get notePlaceholder => translate('notePlaceholder');
  String notePreview(String note) =>
      translate('notePreview', params: {'note': note});

  String get noInternet => translate('noInternet');
  String get noInternetShort => translate('noInternetShort');
  String get checkInternet => translate('checkInternet');
  String get offlineSavePrompt => translate('offlineSavePrompt');
  String get yesSaveDraft => translate('yesSaveDraft');
  String get noCancel => translate('noCancel');
  String get mustBeOnlineForDetails => translate('mustBeOnlineForDetails');
  String get mustBeOnlineToContinue => translate('mustBeOnlineToContinue');

  String get loading => translate('loading');
  String get loadingInvoice => translate('loadingInvoice');
  String get initializingApp => translate('initializingApp');

  String get aboutApp => translate('aboutApp');
  String get developer => translate('developer');
  String get phone => translate('phone');
  String get storeYahya => translate('storeYahya');
  String get storeCommercial => translate('storeCommercial');

  String get dateFormat => translate('dateFormat');
  String invoiceTotal(String total) =>
      translate('invoiceTotal', params: {'total': total});
  String get clearDate => translate('clearDate');
  String get noInvoices => translate('noInvoices');

  String confirmDeleteDraft(String customer) =>
      translate('confirmDeleteDraft', params: {'customer': customer});

  String get notificationMainChannel => translate('notificationMainChannel');
  String get notificationMainDesc => translate('notificationMainDesc');

  String errorUnexpected(String error) =>
      translate('errorUnexpected', params: {'error': error});
  String get errorConnection => translate('errorConnection');
  String get errorTimeout => translate('errorTimeout');
  String errorAppInit(String error) =>
      translate('errorAppInit', params: {'error': error});

  String releaseMessage(String message) =>
      translate('releaseMessage', params: {'message': message});

  String get timeZoneArab => translate('timeZoneArab');
  String get timeZoneArabian => translate('timeZoneArabian');
  String get timeZoneEgypt => translate('timeZoneEgypt');
  String get timeZoneSyria => translate('timeZoneSyria');
  String get timeZoneJordan => translate('timeZoneJordan');

  String get offlineSyncStart => translate('offlineSyncStart');
  String offlineSyncFoundPending(String count) =>
      translate('offlineSyncFoundPending', params: {'count': count});
  String offlineSyncProcessing(String id) =>
      translate('offlineSyncProcessing', params: {'id': id});
  String offlineSyncSuccess(String id) =>
      translate('offlineSyncSuccess', params: {'id': id});
  String offlineSyncFailed(String id, String error) =>
      translate('offlineSyncFailed', params: {'id': id, 'error': error});
  String get offlineSyncComplete => translate('offlineSyncComplete');

  String get dbDeletedTemp => translate('dbDeletedTemp');
  String get dbFileNotFound => translate('dbFileNotFound');

  String get settings => translate('settings');
  String get language => translate('language');
  String get arabic => translate('arabic');
  String get english => translate('english');
  String get saveChanges => translate('saveChanges');
  String get cancel => translate('cancel');
  String get developerLabel => translate('developerLabel');
  String get phoneLabel => translate('phoneLabel');
  String get commercialStoreLabel => translate('commercialStoreLabel');

  String get orders => translate('orders');
  String get warehouse => translate('warehouse');
  String get customers => translate('customers');

  String get offline => translate('offline');
  String get online => translate('online');
  String get edit => translate('edit');
  String get delete => translate('delete');
  String get confirmDeletion => translate('confirmDeletion');
  String get ok => translate('ok');
}

class _AppLocalizationsDelegate
    extends LocalizationsDelegate<AppLocalizations> {
  const _AppLocalizationsDelegate();

  @override
  bool isSupported(Locale locale) {
    // Support English and Arabic
    return ['en', 'ar'].contains(locale.languageCode);
  }

  @override
  Future<AppLocalizations> load(Locale locale) async {
    // AppLocalizations class is where the JSON loading actually runs
    AppLocalizations localizations = AppLocalizations(locale);
    await localizations.load();
    return localizations;
  }

  @override
  bool shouldReload(_AppLocalizationsDelegate old) => false;
}
