/*
© 2025 Sharon Aicler (saichler@gmail.com)

Layer 8 Ecosystem is licensed under the Apache License, Version 2.0.
*/

// SecScan Reference Registry (PRD §12.2): ImageCategory is the only Prime
// Object picked via a Layer8DReferencePicker in this app (ImageGroup's
// category field, §11.2). ImageGroup/ImageRef are never lookupModel
// targets (ReferenceRegistryCompleteness).
(function() {
    'use strict';
    const ref = window.Layer8RefFactory;
    Layer8DReferenceRegistry.register({
        ...ref.simple('ImageCategory', 'categoryId', 'name', 'Category')
    });
})();
